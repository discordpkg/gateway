package discordgateway

import (
	"fmt"
	"strconv"

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

type GatewayResume struct {
	Token          string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int64  `json:"seq"`
}

type GatewayIdentifyProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

type GatewayIdentify struct {
	Token          string      `json:"token"`
	Properties     interface{} `json:"properties"`
	Compress       bool        `json:"compress,omitempty"`
	LargeThreshold uint8       `json:"large_threshold,omitempty"`
	Shard          [2]uint     `json:"shard"`
	Presence       interface{} `json:"presence"`
	Intents        intent.Flag `json:"intents"`
}

// NewResumableShard when a client/shard disconnects for whatever reason,
// you must create a new instance. To utilise the resume logic from discord
// you must use the new client instance below. Note that it is your responsibility
// to make sure you are allowed to resume. eg. resuming after a "invalid session"
// will most likely fail, and you need to create a new shard again.
//
// You must use the newly returned instance. The argument/function input should
// be left for garbage collection.
func NewResumableShard(deadShard *GatewayState) *GatewayState {
	return &GatewayState{
		conf:      deadShard.conf,
		state:     newStateWithSeqNumber(deadShard.SequenceNumber()),
		sessionID: deadShard.sessionID,
	}
}

func NewShardFromPrevious(deadShard *GatewayState) *GatewayState {
	return &GatewayState{
		conf: deadShard.conf,
	}
}

func NewShard(conf *ClientStateConfig) *GatewayState {
	return &GatewayState{
		conf: *conf,
	}
}

type ClientStateConfig struct {
	Token               string
	Intents             intent.Flag
	ShardID             uint
	TotalNumberOfShards uint
	Properties          GatewayIdentifyProperties
}

// GatewayState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type GatewayState struct {
	conf ClientStateConfig
	state
	// TODO: replace state interface with stateClient struct
	//  interface is used to ensure validity, and avoid write dependencies.
	//  The idea is that state can be re-used in a future voice implementation as well

	sessionID            string
	sentResumeOrIdentify atomic.Bool
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
		Token:          gs.conf.Token,
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
	resumePacket := &GatewayResume{
		Token:          gs.conf.Token,
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
