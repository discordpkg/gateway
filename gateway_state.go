package discordgateway

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/opcode"
)

type CloseError struct {
	Code   uint
	Reason string
}

func (c *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", c.Code, c.Reason)
}

type GatewayHello struct {
	HeartbeatIntervalMilli int64 `json:"heartbeat_interval"`
}

type GatewayReady struct {
	SessionID string `json:"session_id"`
}

type GatewayResume struct {
	BotToken       string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int64  `json:"seq"`
}

type GatewayIdentifyProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

type GatewayIdentify struct {
	BotToken       string      `json:"token"`
	Properties     interface{} `json:"properties"`
	Compress       bool        `json:"compress,omitempty"`
	LargeThreshold uint8       `json:"large_threshold,omitempty"`
	Shard          [2]uint     `json:"shard"`
	Presence       interface{} `json:"presence"`
	Intents        intent.Flag `json:"intents"`
}

func NewGatewayClient(conf *GatewayStateConfig) *GatewayState {
	return &GatewayState{
		conf:  *conf,
		state: newState(),
	}
}

type GatewayStateConfig struct {
	BotToken            string
	Intents             intent.Flag
	ShardID             uint
	TotalNumberOfShards uint
	Properties          GatewayIdentifyProperties
}

// GatewayState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type GatewayState struct {
	conf GatewayStateConfig
	state
	// TODO: replace state interface with stateClient struct
	//  interface is used to ensure validity, and avoid write dependencies.
	//  The idea is that state can be re-used in a future voice implementation as well

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

	if payload.EventFlag == event.Ready {
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
		Intents:        gs.conf.Intents,
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
