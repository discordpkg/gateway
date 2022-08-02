package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/internal/util"
	"github.com/discordpkg/gateway/json"
	"go.uber.org/atomic"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"strconv"
	"strings"

	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/opcode"
)

const (
	NormalCloseCode  uint16 = 1000
	RestartCloseCode uint16 = 1012
)

type GatewayPayload struct {
	Op        opcode.Type     `json:"op"`
	Data      json.RawMessage `json:"d"`
	Seq       int64           `json:"s,omitempty"`
	EventName event.Type      `json:"t,omitempty"`
	Outdated  bool            `json:"-"`
}

var ErrSequenceNumberSkipped = errors.New("the sequence number increased with more than 1, events lost")

type DiscordError struct {
	CloseCode closecode.Type
	OpCode    opcode.Type
	Reason    string
}

func (c *DiscordError) Error() string {
	return fmt.Sprintf("[%d | %d]: %s", c.CloseCode, c.OpCode, c.Reason)
}

func (c DiscordError) CanReconnect() bool {
	return closecode.CanReconnectAfter(c.CloseCode) || opcode.CanReconnectAfter(c.OpCode)
}

func NewGatewayState(botToken string, options ...Option) (*GatewayState, error) {
	gs := &GatewayState{
		botToken: botToken,
	}

	for i := range options {
		if err := options[i](gs); err != nil {
			return nil, err
		}
	}

	if gs.intents == 0 && (len(gs.guildEvents) > 0 || len(gs.directMessageEvents) > 0) {
		// derive intents
		gs.intents |= intent.GuildEventsToIntents(gs.guildEvents)
		gs.intents |= intent.DMEventsToIntents(gs.directMessageEvents)

		// whitelisted events specified events only
		gs.whitelist = util.Set[event.Type]{}
		gs.whitelist.Add(gs.guildEvents...)
		gs.whitelist.Add(gs.directMessageEvents...)

		// crucial for normal function
		gs.whitelist.Add(event.Ready, event.Resumed)
	}

	// rate limits
	if gs.commandRateLimitChan == nil {
		var closer io.Closer
		gs.commandRateLimitChan, closer = NewCommandRateLimiter()
		gs.closers = append(gs.closers, closer)
	}
	if gs.identifyRateLimiter == nil {
		channel, closer := NewIdentifyRateLimiter()
		gs.closers = append(gs.closers, closer)

		gs.identifyRateLimiter = &channelTaker{c: channel}
	}

	// connection properties
	if gs.connectionProperties == nil {
		gs.connectionProperties = &IdentifyConnectionProperties{
			OS:      runtime.GOOS,
			Browser: "github.com/discordpkg/gateway",
			Device:  "github.com/discordpkg/gateway",
		}
	}

	// sharding
	if gs.totalNumberOfShards == 0 {
		if gs.shardID == 0 {
			gs.totalNumberOfShards = 1
		} else {
			return nil, errors.New("missing shard count")
		}
	}
	if uint(gs.shardID) > gs.totalNumberOfShards {
		return nil, errors.New("shard id is higher than shard count")
	}

	return gs, nil
}

// GatewayState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type GatewayState struct {
	sequenceNumber atomic.Int64
	closed         atomic.Bool

	// events that are not found in the whitelist are viewed as redundant and are
	// skipped / ignored
	whitelist           util.Set[event.Type]
	directMessageEvents []event.Type
	guildEvents         []event.Type

	intents intent.Type

	sessionID            string
	sentResumeOrIdentify atomic.Bool
	closers              []io.Closer

	shardID              ShardID
	totalNumberOfShards  uint
	connectionProperties *IdentifyConnectionProperties
	commandRateLimitChan <-chan int
	identifyRateLimiter  IdentifyRateLimiter
	botToken             string
}

func (gs *GatewayState) String() string {
	data := ""
	data += fmt.Sprintln("device:", gs.connectionProperties.Device)
	data += fmt.Sprintln(fmt.Sprintf("shard: %d / %d", gs.shardID, gs.totalNumberOfShards))
	data += fmt.Sprintln("intents:", gs.intents)
	data += fmt.Sprintln("events:", gs.intents)
	return data
}

func (gs *GatewayState) SequenceNumber() int64 {
	return gs.sequenceNumber.Load()
}

func (gs *GatewayState) Closed() bool {
	return gs.closed.Load()
}

func (gs *GatewayState) WriteNormalClose(client io.Writer) error {
	return gs.writeClose(client, NormalCloseCode)
}

func (gs *GatewayState) WriteRestartClose(client io.Writer) error {
	return gs.writeClose(client, RestartCloseCode)
}

func (gs *GatewayState) writeClose(client io.Writer, code uint16) error {
	writeIfOpen := func() error {
		if gs.closed.CAS(false, true) {
			closeCodeBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(closeCodeBuf, code)

			_, err := client.Write(closeCodeBuf)
			return err
		}
		return net.ErrClosed
	}

	if err := writeIfOpen(); err != nil {
		if !errors.Is(err, net.ErrClosed) && strings.Contains(err.Error(), "use of closed connection") {
			return net.ErrClosed
		}
		return err
	}
	return nil
}

func (gs *GatewayState) Close() error {
	for _, closer := range gs.closers {
		_ = closer.Close()
	}
	return nil
}

func (gs *GatewayState) Write(client io.Writer, opc command.Type, payload json.RawMessage) (err error) {
	if gs.closed.Load() {
		return net.ErrClosed
	}

	if opc != command.Heartbeat {
		// heartbeat should always be sent, regardless!
		<-gs.commandRateLimitChan
	}
	if opc == command.Identify {
		if available := gs.identifyRateLimiter.Take(gs.shardID); !available {
			return errors.New("identify rate limiter denied shard to identify")
		}
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
		Op:   opcode.Type(opc),
		Data: payload,
	}

	var data []byte
	data, err = json.Marshal(&packet)
	if err != nil {
		return fmt.Errorf("unable to marshal packet; %w", err)
	}

	_, err = client.Write(data)
	return err
}

func (gs *GatewayState) Read(client io.Reader) (*GatewayPayload, int, error) {
	if gs.closed.Load() {
		return nil, 0, net.ErrClosed
	}

	data, err := ioutil.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &GatewayPayload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	prevSeq := gs.sequenceNumber.Load()
	packet.Outdated = prevSeq >= packet.Seq
	if packet.Seq-prevSeq > 1 {
		return nil, 0, ErrSequenceNumberSkipped
	}
	if !packet.Outdated {
		gs.sequenceNumber.Store(packet.Seq)
	}

	if packet.EventName == event.Ready {
		// need to store session ID for resume
		ready := Ready{}
		if err := json.Unmarshal(packet.Data, &ready); err != nil || ready.SessionID == "" {
			return packet, len(data), fmt.Errorf("failed to extract session id from ready event. %w", err)
		}
		gs.sessionID = ready.SessionID
	}
	return packet, len(data), nil
}

func (gs *GatewayState) ProcessNextMessage(pipe io.Reader, textWriter, closeWriter io.Writer) (payload *GatewayPayload, redundant bool, err error) {
	payload, _, err = gs.Read(pipe)
	if errors.Is(err, ErrSequenceNumberSkipped) {
		_ = gs.WriteRestartClose(closeWriter)
		return nil, true, err
	}
	if err != nil {
		return nil, false, err
	}

	redundant, err = gs.ProcessPayload(payload, textWriter, closeWriter)
	return payload, redundant, err
}

func (gs *GatewayState) ProcessPayload(payload *GatewayPayload, textWriter, closeWriter io.Writer) (redundant bool, err error) {
	switch payload.Op {
	case opcode.Heartbeat:
		if err := gs.Heartbeat(textWriter); err != nil {
			return false, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.Hello:
		if gs.HaveIdentified() {
			return true, nil
		}
		if gs.HaveSessionID() {
			if err := gs.Resume(textWriter); err != nil {
				return false, fmt.Errorf("sending resume failed. closing. %w", err)
			}
		} else {
			if err := gs.Identify(textWriter); err != nil {
				return false, fmt.Errorf("identify failed. closing. %w", err)
			}
		}
	case opcode.Dispatch:
		if !gs.EventIsWhitelisted(payload.EventName) {
			return true, nil
		}
	case opcode.InvalidSession:
		gs.InvalidateSession(closeWriter)
	case opcode.Reconnect:
		_ = gs.WriteRestartClose(closeWriter)
	default:
		// unhandled operation code
		// TODO: logging?
	}

	return false, nil
}

// ProcessCloseCode process close code sent by discord
func (gs *GatewayState) ProcessCloseCode(code closecode.Type, reason string, closeWriter io.Writer) error {
	switch code {
	case closecode.ClientReconnecting, closecode.UnknownError:
		// allow resume
	default:
		gs.InvalidateSession(closeWriter)
	}
	return &DiscordError{CloseCode: code, Reason: reason}
}

// Heartbeat Close method may be used if Write fails
func (gs *GatewayState) Heartbeat(client io.Writer) error {
	seq := gs.SequenceNumber()
	seqStr := strconv.FormatInt(seq, 10)
	return gs.Write(client, command.Heartbeat, []byte(seqStr))
}

// Identify Close method may be used if Write fails
func (gs *GatewayState) Identify(client io.Writer) error {
	identifyPacket := &Identify{
		BotToken:       gs.botToken,
		Properties:     &gs.connectionProperties,
		Compress:       false,
		LargeThreshold: 0,
		Shard:          [2]uint{uint(gs.shardID), gs.totalNumberOfShards},
		Presence:       nil,
		Intents:        gs.intents,
	}

	data, err := json.Marshal(identifyPacket)
	if err != nil {
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if err = gs.Write(client, command.Identify, data); err != nil {
		return err
	}

	gs.sentResumeOrIdentify.Store(true)
	return nil
}

// Resume Close method may be used if Write fails
func (gs *GatewayState) Resume(client io.Writer) error {
	if !gs.HaveSessionID() {
		return errors.New("missing session id, can not resume connection")
	}
	resumePacket := &Resume{
		BotToken:       gs.botToken,
		SessionID:      gs.sessionID,
		SequenceNumber: gs.SequenceNumber(),
	}
	data, err := json.Marshal(&resumePacket)
	if err != nil {
		return fmt.Errorf("unable to marshal resume payload. %w", err)
	}

	if err = gs.Write(client, command.Resume, data); err != nil {
		return err
	}

	gs.sentResumeOrIdentify.Store(true)
	return nil
}

func (gs *GatewayState) SessionID() string {
	return gs.sessionID
}

func (gs *GatewayState) HaveSessionID() bool {
	return gs.sessionID != ""
}

func (gs *GatewayState) HaveIdentified() bool {
	return gs.sentResumeOrIdentify.Load()
}

func (gs *GatewayState) InvalidateSession(closeWriter io.Writer) {
	if err := gs.WriteNormalClose(closeWriter); err != nil && !errors.Is(err, net.ErrClosed) {
		// TODO: so what?
	}
	gs.sessionID = ""
	//gs.state = nil
}

func (gs *GatewayState) EventIsWhitelisted(evt event.Type) bool {
	if gs.whitelist != nil {
		return gs.whitelist.Contains(evt)
	}

	return true
}
