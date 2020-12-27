package discordgateway

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/intent"
)

type CloseError struct {
	Code   uint
	Reason string
}

func (c *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", c.Code, c.Reason)
}

type GatewayPayload struct {
	Op        uint8      `json:"op"`
	Data      RawMessage `json:"d"`
	Seq       int64      `json:"s,omitempty"`
	EventName string     `json:"t,omitempty"`
	// EventID   event.Flag `json:"-"`
	Outdated bool `json:"-"`
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
func NewResumableShard(deadShard *ClientState) *ClientState {
	return &ClientState{
		conf:           deadShard.conf,
		sessionID:      deadShard.sessionID,
		sequenceNumber: deadShard.sequenceNumber,
	}
}

func NewShardFromPrevious(deadShard *ClientState) *ClientState {
	return &ClientState{
		conf: deadShard.conf,
	}
}

func NewShard(conf *ClientStateConfig) *ClientState {
	return &ClientState{
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

// ClientState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type ClientState struct {
	conf ClientStateConfig

	sequenceNumber atomic.Int64
	sessionID      string

	stateClosed          atomic.Bool
	sentResumeOrIdentify atomic.Bool
}

func (c *ClientState) Closed() bool {
	return c.stateClosed.Load()
}

func (c *ClientState) WriteClose(client IOFlushWriter) error {
	if c.stateClosed.CAS(false, true) {
		// there is no real benefit to give Discord a reason.
		// relevant errors should instead be logged for diagnostic purposes.
		closeCodeBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(closeCodeBuf, uint16(1000))

		if _, err := client.Write(closeCodeBuf); err != nil {
			return fmt.Errorf("unable to write close code to Discord. %w", err)
		}
		return client.Flush()
	}
	return io.ErrClosedPipe
}

// Read until a new data frame with updated info comes in, or fails.
func (c *ClientState) Read(client IOReader) (*GatewayPayload, int, error) {
	if c.stateClosed.Load() {
		return nil, 0, io.ErrClosedPipe
	}

	data, err := ioutil.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &GatewayPayload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	prevSeq := c.sequenceNumber.Load()
	packet.Outdated = prevSeq >= packet.Seq
	if !packet.Outdated {
		c.sequenceNumber.Store(packet.Seq)
	}
	if packet.Seq-prevSeq > 1 {
		// TODO: log skip
	}

	return packet, len(data), nil
}

// write Close method may be used if Write fails
func (c *ClientState) write(client IOFlushWriter, opCode uint8, payload RawMessage) (err error) {
	if c.stateClosed.Load() {
		return io.ErrClosedPipe
	}

	defer func() {
		if err != nil {
			// TODO: skip error wrapping if the connection was closed before hand
			//  no need to close twice and pretend this was the first place to
			//  do so..
			// _ = client.Close()
			err = fmt.Errorf("client after failed dispatch. %w", err)
		}
	}()

	packet := GatewayPayload{
		Op:   opCode,
		Data: payload,
	}

	var data []byte
	data, err = json.Marshal(&packet)
	if err != nil {
		return fmt.Errorf("unable to marshal packet; %w", err)
	}

	var length int
	length, err = client.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send packet; %w", err)
	}
	if length == 0 {
		return errors.New("no data was sent")
	}

	return client.Flush()
}

// Heartbeat Close method may be used if Write fails
func (c *ClientState) Heartbeat(client IOFlushWriter) error {
	seq := c.sequenceNumber.Load()
	seqStr := strconv.FormatInt(seq, 10)
	return c.write(client, 1, []byte(seqStr))
}

// Identify Close method may be used if Write fails
func (c *ClientState) Identify(client IOFlushWriter) error {
	identifyPacket := &GatewayIdentify{
		Token:          c.conf.Token,
		Properties:     &c.conf.Properties,
		Compress:       false,
		LargeThreshold: 0,
		Shard:          [2]uint{c.conf.ShardID, c.conf.TotalNumberOfShards},
		Presence:       nil,
		Intents:        c.conf.Intents,
	}

	data, err := json.Marshal(identifyPacket)
	if err != nil {
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if err = c.write(client, 2, data); err != nil {
		return err
	}

	c.sentResumeOrIdentify.Store(true)
	return nil
}

// Resume Close method may be used if Write fails
func (c *ClientState) Resume(client IOFlushWriter) error {
	resumePacket := &GatewayResume{
		Token:          c.conf.Token,
		SessionID:      c.sessionID,
		SequenceNumber: c.sequenceNumber.Load(),
	}
	data, err := json.Marshal(&resumePacket)
	if err != nil {
		return fmt.Errorf("unable to marshal resume payload. %w", err)
	}

	if err = c.write(client, 6, data); err != nil {
		return err
	}

	c.sentResumeOrIdentify.Store(true)
	return nil
}

func (c *ClientState) HaveSessionID() bool {
	return c.sessionID != ""
}

func (c *ClientState) HaveIdentified() bool {
	return c.sentResumeOrIdentify.Load()
}
