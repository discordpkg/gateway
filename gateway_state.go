package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/internal/util"
	"github.com/discordpkg/gateway/json"
	"go.uber.org/atomic"
	"io"
	"net"
	"runtime"
	"strconv"

	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/opcode"
)

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

	gs.state = newStateWithSeqNumber(gs.initialSequenceNumber)
	return gs, nil
}

// GatewayState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type GatewayState struct {
	state
	// TODO: replace state interface with stateClient struct
	//  interface is used to ensure validity, and avoid write dependencies.
	//  The idea is that state can be re-used in a future voice implementation as well

	// events that are not found in the whitelist are viewed as redundant and are
	// skipped / ignored
	whitelist           util.Set[event.Type]
	directMessageEvents []event.Type
	guildEvents         []event.Type

	intents intent.Type

	initialSequenceNumber int64
	sessionID             string
	sentResumeOrIdentify  atomic.Bool
	closers               []io.Closer

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

func (gs *GatewayState) Close() error {
	for _, closer := range gs.closers {
		_ = closer.Close()
	}
	return nil
}

func (gs *GatewayState) Write(client io.Writer, opc command.Type, payload json.RawMessage) (err error) {
	if opc != command.Heartbeat {
		// heartbeat should always be sent, regardless!
		<-gs.commandRateLimitChan
	}
	if opc == command.Identify {
		if available := gs.identifyRateLimiter.Take(gs.shardID); !available {
			return errors.New("identify rate limiter denied shard to identify")
		}
	}

	return gs.state.Write(client, opc, payload)
}

func (gs *GatewayState) Read(client io.Reader) (*GatewayPayload, int, error) {
	payload, length, err := gs.state.Read(client)
	if err != nil {
		return nil, 0, err
	}

	if payload.EventName == event.Ready {
		// need to store session ID for resume
		ready := Ready{}
		if err := json.Unmarshal(payload.Data, &ready); err != nil || ready.SessionID == "" {
			return payload, length, fmt.Errorf("failed to extract session id from ready event. %w", err)
		}
		gs.sessionID = ready.SessionID
	}
	return payload, length, nil
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
