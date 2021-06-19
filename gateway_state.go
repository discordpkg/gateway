package discordgateway

import (
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"
	"github.com/bradfitz/iter"
	"io"
	"net"
	"strconv"
	"time"

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

type channelCloser struct {
	c      chan int
	closed atomic.Bool
}

func (c *channelCloser) Close() error {
	if c.c != nil {
		close(c.c)
		c.closed.Store(true)
	}
	return nil
}

func NewCommandRateLimiter() (<-chan int, io.Closer) {
	burstSize := 120
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return NewRateLimiter(burstSize, 60*time.Second)
}

func NewIdentifyRateLimiter() (<-chan int, io.Closer) {
	return NewRateLimiter(1, 5*time.Second)
}

func NewRateLimiter(burstCapacity int, burstDuration time.Duration) (<-chan int, io.Closer) {
	c := make(chan int, burstCapacity)
	closer := &channelCloser{c: c}

	go func() {
		refill := func() {
			burstSize := burstCapacity - len(c)

			for range iter.N(burstSize) {
				c <- 0
			}
		}

		t := time.NewTicker(burstDuration)
		defer t.Stop()

		refill()
		for {
			<-t.C

			if closer.closed.Load() {
				// channel has been closed
				break
			}

			refill()
		}
	}()

	return c, closer
}

var emptyStruct struct{}

func NewGatewayClient(conf *GatewayStateConfig) *GatewayState {
	gs := &GatewayState{
		conf:      *conf,
		state:     newState(),
		whitelist: make(map[event.Type]struct{}),
	}

	// derive intents
	gs.intents = gs.conf.Intents()

	// whitelisted events
	gs.whitelist = gs.conf.EventsMap()
	gs.whitelist[event.Ready] = emptyStruct
	gs.whitelist[event.Resumed] = emptyStruct

	// rate limits
	if gs.conf.CommandRateLimitChan == nil {
		var closer io.Closer
		gs.conf.CommandRateLimitChan, closer = NewCommandRateLimiter()
		gs.closers = append(gs.closers, closer)
	}
	if gs.conf.IdentifyRateLimitChan == nil {
		var closer io.Closer
		gs.conf.IdentifyRateLimitChan, closer = NewIdentifyRateLimiter()
		gs.closers = append(gs.closers, closer)
	}

	return gs
}

type GatewayStateConfig struct {
	BotToken              string
	ShardID               ShardID
	TotalNumberOfShards   uint
	Properties            GatewayIdentifyProperties
	CommandRateLimitChan  <-chan int
	IdentifyRateLimitChan <-chan int
	GuildEvents           []event.Type
	DMEvents              []event.Type
}

func (gsc *GatewayStateConfig) Intents() (intents intent.Type) {
	intents |= intent.GuildEventsToIntents(gsc.GuildEvents)
	intents |= intent.DMEventsToIntents(gsc.DMEvents)
	return intents
}

func (gsc *GatewayStateConfig) EventsMap() map[event.Type]struct{} {
	events := make(map[event.Type]struct{})
	for _, evt := range gsc.DMEvents {
		events[evt] = emptyStruct
	}
	for _, evt := range gsc.GuildEvents {
		events[evt] = emptyStruct
	}
	return events
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
	closers              []io.Closer
}

func (gs *GatewayState) Close() error {
	for _, closer := range gs.closers {
		_ = closer.Close()
	}
	return nil
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

	if op != opcode.EventHeartbeat {
		// heartbeat should always be sent, regardless!
		<-gs.conf.CommandRateLimitChan
	}
	if op == opcode.EventIdentify {
		<-gs.conf.IdentifyRateLimitChan
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
		Shard:          [2]uint{uint(gs.conf.ShardID), gs.conf.TotalNumberOfShards},
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
