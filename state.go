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

func NewState(botToken string, options ...Option) (*State, error) {
	st := &State{
		botToken: botToken,
	}

	for i := range options {
		if err := options[i](st); err != nil {
			return nil, err
		}
	}

	if st.intents == 0 && (len(st.guildEvents) > 0 || len(st.directMessageEvents) > 0) {
		// derive intents
		st.intents |= intent.GuildEventsToIntents(st.guildEvents)
		st.intents |= intent.DMEventsToIntents(st.directMessageEvents)

		// whitelisted events specified events only
		st.whitelist = util.Set[event.Type]{}
		st.whitelist.Add(st.guildEvents...)
		st.whitelist.Add(st.directMessageEvents...)

		// crucial for normal function
		st.whitelist.Add(event.Ready, event.Resumed)
	}

	// rate limits
	if st.commandRateLimitChan == nil {
		var closer io.Closer
		st.commandRateLimitChan, closer = NewCommandRateLimiter()
		st.closers = append(st.closers, closer)
	}
	if st.identifyRateLimiter == nil {
		channel, closer := NewIdentifyRateLimiter()
		st.closers = append(st.closers, closer)

		st.identifyRateLimiter = &channelTaker{c: channel}
	}

	// connection properties
	if st.connectionProperties == nil {
		st.connectionProperties = &IdentifyConnectionProperties{
			OS:      runtime.GOOS,
			Browser: "github.com/discordpkg/gateway",
			Device:  "github.com/discordpkg/gateway",
		}
	}

	// sharding
	if st.totalNumberOfShards == 0 {
		if st.shardID == 0 {
			st.totalNumberOfShards = 1
		} else {
			return nil, errors.New("missing shard count")
		}
	}
	if uint(st.shardID) > st.totalNumberOfShards {
		return nil, errors.New("shard id is higher than shard count")
	}

	return st, nil
}

// State should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type State struct {
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

func (st *State) String() string {
	data := ""
	data += fmt.Sprintln("device:", st.connectionProperties.Device)
	data += fmt.Sprintln(fmt.Sprintf("shard: %d / %d", st.shardID, st.totalNumberOfShards))
	data += fmt.Sprintln("intents:", st.intents)
	data += fmt.Sprintln("events:", st.intents)
	return data
}

func (st *State) SequenceNumber() int64 {
	return st.sequenceNumber.Load()
}

func (st *State) Closed() bool {
	return st.closed.Load()
}

func (st *State) WriteNormalClose(client io.Writer) error {
	return st.writeClose(client, NormalCloseCode)
}

func (st *State) WriteRestartClose(client io.Writer) error {
	return st.writeClose(client, RestartCloseCode)
}

func (st *State) writeClose(client io.Writer, code uint16) error {
	writeIfOpen := func() error {
		if st.closed.CAS(false, true) {
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

func (st *State) Close() error {
	for _, closer := range st.closers {
		_ = closer.Close()
	}
	return nil
}

func (st *State) Write(client io.Writer, opc command.Type, payload json.RawMessage) (err error) {
	if st.closed.Load() {
		return net.ErrClosed
	}

	if opc != command.Heartbeat {
		// heartbeat should always be sent, regardless!
		<-st.commandRateLimitChan
	}
	if opc == command.Identify {
		if available := st.identifyRateLimiter.Take(st.shardID); !available {
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

func (st *State) Read(client io.Reader) (*GatewayPayload, int, error) {
	if st.closed.Load() {
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

	prevSeq := st.sequenceNumber.Load()
	packet.Outdated = prevSeq >= packet.Seq
	if packet.Seq-prevSeq > 1 {
		return nil, 0, ErrSequenceNumberSkipped
	}
	if !packet.Outdated {
		st.sequenceNumber.Store(packet.Seq)
	}

	if packet.EventName == event.Ready {
		// need to store session ID for resume
		ready := Ready{}
		if err := json.Unmarshal(packet.Data, &ready); err != nil || ready.SessionID == "" {
			return packet, len(data), fmt.Errorf("failed to extract session id from ready event. %w", err)
		}
		st.sessionID = ready.SessionID
	}
	return packet, len(data), nil
}

func (st *State) ProcessNextMessage(pipe io.Reader, textWriter, closeWriter io.Writer) (payload *GatewayPayload, redundant bool, err error) {
	payload, _, err = st.Read(pipe)
	if errors.Is(err, ErrSequenceNumberSkipped) {
		_ = st.WriteRestartClose(closeWriter)
		return nil, true, err
	}
	if err != nil {
		return nil, false, err
	}

	redundant, err = st.ProcessPayload(payload, textWriter, closeWriter)
	return payload, redundant, err
}

func (st *State) ProcessPayload(payload *GatewayPayload, textWriter, closeWriter io.Writer) (redundant bool, err error) {
	switch payload.Op {
	case opcode.Heartbeat:
		if err := st.Heartbeat(textWriter); err != nil {
			return false, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.Hello:
		if st.HaveIdentified() {
			return true, nil
		}
		if st.HaveSessionID() {
			if err := st.Resume(textWriter); err != nil {
				return false, fmt.Errorf("sending resume failed. closing. %w", err)
			}
		} else {
			if err := st.Identify(textWriter); err != nil {
				return false, fmt.Errorf("identify failed. closing. %w", err)
			}
		}
	case opcode.Dispatch:
		if !st.EventIsWhitelisted(payload.EventName) {
			return true, nil
		}
	case opcode.InvalidSession:
		st.InvalidateSession(closeWriter)
	case opcode.Reconnect:
		_ = st.WriteRestartClose(closeWriter)
	default:
		// unhandled operation code
		// TODO: logging?
	}

	return false, nil
}

// ProcessCloseCode process close code sent by discord
func (st *State) ProcessCloseCode(code closecode.Type, reason string, closeWriter io.Writer) error {
	switch code {
	case closecode.ClientReconnecting, closecode.UnknownError:
		// allow resume
	default:
		st.InvalidateSession(closeWriter)
	}
	return &DiscordError{CloseCode: code, Reason: reason}
}

// Heartbeat Close method may be used if Write fails
func (st *State) Heartbeat(client io.Writer) error {
	seq := st.SequenceNumber()
	seqStr := strconv.FormatInt(seq, 10)
	return st.Write(client, command.Heartbeat, []byte(seqStr))
}

// Identify Close method may be used if Write fails
func (st *State) Identify(client io.Writer) error {
	identifyPacket := &Identify{
		BotToken:       st.botToken,
		Properties:     &st.connectionProperties,
		Compress:       false,
		LargeThreshold: 0,
		Shard:          [2]uint{uint(st.shardID), st.totalNumberOfShards},
		Presence:       nil,
		Intents:        st.intents,
	}

	data, err := json.Marshal(identifyPacket)
	if err != nil {
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if err = st.Write(client, command.Identify, data); err != nil {
		return err
	}

	st.sentResumeOrIdentify.Store(true)
	return nil
}

// Resume Close method may be used if Write fails
func (st *State) Resume(client io.Writer) error {
	if !st.HaveSessionID() {
		return errors.New("missing session id, can not resume connection")
	}
	resumePacket := &Resume{
		BotToken:       st.botToken,
		SessionID:      st.sessionID,
		SequenceNumber: st.SequenceNumber(),
	}
	data, err := json.Marshal(&resumePacket)
	if err != nil {
		return fmt.Errorf("unable to marshal resume payload. %w", err)
	}

	if err = st.Write(client, command.Resume, data); err != nil {
		return err
	}

	st.sentResumeOrIdentify.Store(true)
	return nil
}

func (st *State) SessionID() string {
	return st.sessionID
}

func (st *State) HaveSessionID() bool {
	return st.sessionID != ""
}

func (st *State) HaveIdentified() bool {
	return st.sentResumeOrIdentify.Load()
}

func (st *State) InvalidateSession(closeWriter io.Writer) {
	if err := st.WriteNormalClose(closeWriter); err != nil && !errors.Is(err, net.ErrClosed) {
		// TODO: so what?
	}
	st.sessionID = ""
	//gs.state = nil
}

func (st *State) EventIsWhitelisted(evt event.Type) bool {
	if st.whitelist != nil {
		return st.whitelist.Contains(evt)
	}

	return true
}
