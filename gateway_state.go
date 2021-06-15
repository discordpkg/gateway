package discordgateway

import (
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"
	"net"
	"strconv"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/opcode"
)

var NormalCloseErr = &CloseError{Code: 1000, Reason: "client is going away"}

type CloseError struct {
	Code   uint
	Reason string
}

func (c *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", c.Code, c.Reason)
}

var emptyStruct struct{}

func NewGatewayClient(conf *GatewayStateConfig) *GatewayState {
	gs := &GatewayState{
		conf:      *conf,
		state:     newState(),
		whitelist: make(map[event.Type]struct{}),
	}

	// derive intents
	gs.intents = intent.Merge(conf.Intents()...)

	// whitelist events
	for _, evt := range conf.DMEvents {
		gs.whitelist[evt] = emptyStruct
	}
	for _, evt := range conf.GuildEvents {
		gs.whitelist[evt] = emptyStruct
	}

	return gs
}

type GatewayStateConfig struct {
	BotToken            string
	ShardID             uint
	TotalNumberOfShards uint
	Properties          GatewayIdentifyProperties
	GuildEvents         []event.Type
	DMEvents            []event.Type
}

func (gsc *GatewayStateConfig) Intents() (intents []intent.Type) {
	intentsMap := make(map[intent.Type]struct{})
	for _, i := range intent.DMEventsToIntents(gsc.DMEvents) {
		intentsMap[i] = emptyStruct
	}
	for _, i := range intent.GuildEventsToIntents(gsc.GuildEvents) {
		intentsMap[i] = emptyStruct
	}

	for i := range intentsMap {
		intents = append(intents, i)
	}
	return intents
}

// GatewayState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type GatewayState struct {
	conf GatewayStateConfig
	state
	// TODO: replace state interface with stateClient struct
	//  interface is used to ensure validity, and avoid write dependencies.
	//  The idea is that state can be re-used in a future voice implementation as well

	// events that are not found in the whitelist are viewed as redundant and are
	// skipped / ignored
	whitelist            map[event.Type]struct{}
	intents              intent.Type
	sessionID            string
	sentResumeOrIdentify atomic.Bool
}

func (gs *GatewayState) isSendOpCode(op opcode.OpCode) bool {
	validOps := []opcode.OpCode{
		opcode.EventHeartbeat, opcode.EventIdentify,
		opcode.EventPresenceUpdate, opcode.EventVoiceStateUpdate,
		opcode.EventResume, opcode.EventRequestGuildMembers,
	}

	for _, validOp := range validOps {
		if op == validOp {
			return true
		}
	}
	return false
}

func (gs *GatewayState) Write(client IOFlushWriter, op opcode.OpCode, payload json.RawMessage) (err error) {
	if !gs.isSendOpCode(op) {
		return errors.New(fmt.Sprintf("operation code %d is not for outgoing payloads", op))
	}

	return gs.state.Write(client, op, payload)
}

func (gs *GatewayState) Read(client IOReader) (*GatewayPayload, int, error) {
	payload, length, err := gs.state.Read(client)
	if err != nil {
		return nil, 0, err
	}

	if payload.EventName == event.Ready {
		// need to store session ID for resume
		ready := GatewayReady{}
		if err := json.Unmarshal(payload.Data, &ready); err != nil || ready.SessionID == "" {
			return payload, length, fmt.Errorf("failed to extract session id from ready event. %w", err)
		}
		gs.sessionID = ready.SessionID
	}
	return payload, length, nil
}

// Heartbeat Close method may be used if Write fails
func (gs *GatewayState) Heartbeat(client IOFlushWriter) error {
	seq := gs.SequenceNumber()
	seqStr := strconv.FormatInt(seq, 10)
	return gs.Write(client, opcode.EventHeartbeat, []byte(seqStr))
}

// Identify Close method may be used if Write fails
func (gs *GatewayState) Identify(client IOFlushWriter) error {
	identifyPacket := &GatewayIdentify{
		BotToken:       gs.conf.BotToken,
		Properties:     &gs.conf.Properties,
		Compress:       false,
		LargeThreshold: 0,
		Shard:          [2]uint{gs.conf.ShardID, gs.conf.TotalNumberOfShards},
		Presence:       nil,
		Intents:        gs.intents,
	}

	data, err := json.Marshal(identifyPacket)
	if err != nil {
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if err = gs.Write(client, opcode.EventIdentify, data); err != nil {
		return err
	}

	gs.sentResumeOrIdentify.Store(true)
	return nil
}

// Resume Close method may be used if Write fails
func (gs *GatewayState) Resume(client IOFlushWriter) error {
	if !gs.HaveSessionID() {
		return errors.New("missing session id, can not resume connection")
	}
	resumePacket := &GatewayResume{
		BotToken:       gs.conf.BotToken,
		SessionID:      gs.sessionID,
		SequenceNumber: gs.SequenceNumber(),
	}
	data, err := json.Marshal(&resumePacket)
	if err != nil {
		return fmt.Errorf("unable to marshal resume payload. %w", err)
	}

	if err = gs.Write(client, opcode.EventResume, data); err != nil {
		return err
	}

	gs.sentResumeOrIdentify.Store(true)
	return nil
}

func (gs *GatewayState) HaveSessionID() bool {
	return gs.sessionID != ""
}

func (gs *GatewayState) HaveIdentified() bool {
	return gs.sentResumeOrIdentify.Load()
}

func (gs *GatewayState) InvalidateSession(closeWriter IOFlushCloseWriter) {
	if err := gs.WriteNormalClose(closeWriter); err != nil && !errors.Is(err, net.ErrClosed) {
		// TODO: so what?
	}
	gs.sessionID = ""
	//gs.state = nil
}

func (gs *GatewayState) DemultiplexEvent(payload *GatewayPayload, writer IOFlushWriter) (redundant bool, err error) {
	switch payload.Op {
	case opcode.EventHeartbeat:
		if err := gs.Heartbeat(writer); err != nil {
			return false, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.EventHello:
		if gs.HaveIdentified() {
			return true, nil
		}
		if gs.HaveSessionID() {
			if err := gs.Resume(writer); err != nil {
				return false, fmt.Errorf("sending resume failed. closing. %w", err)
			}
		} else {
			if err := gs.Identify(writer); err != nil {
				return false, fmt.Errorf("identify failed. closing. %w", err)
			}
		}
		return false, nil
	case opcode.EventDispatch:
		if _, whitelisted := gs.whitelist[payload.EventName]; whitelisted {
			return false, nil
		}
	case opcode.EventHeartbeatACK, opcode.EventInvalidSession, opcode.EventReconnect:
		return false, nil
	default:
		// TODO: log new unhandled operation code
	}

	return true, nil
}
