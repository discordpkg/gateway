package discordgateway

import (
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/command"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/andersfylling/discordgateway/closecode"
	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"
	"github.com/bradfitz/iter"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/opcode"
)

type CloseError struct {
	Code   closecode.Type
	Reason string
}

func (c *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", c.Code, c.Reason)
}

type channelTaker struct {
	c <-chan int
}

func (c *channelTaker) Take(_ ShardID) bool {
	if c.c != nil {
		select {
		case _, ok := <-c.c:
			if ok {
				return true
			}
		}
	}
	return false
}

type channelCloser struct {
	mu     sync.Mutex
	c      chan int
	closed bool
}

func (c *channelCloser) Closed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *channelCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c != nil {
		close(c.c)
		c.closed = true
	}
	return nil
}

func NewCommandRateLimiter() (<-chan int, io.Closer) {
	burstSize, duration := 120, 60*time.Second
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return NewRateLimiter(burstSize, duration)
}

func NewIdentifyRateLimiter() (<-chan int, io.Closer) {
	return NewRateLimiter(1, 5*time.Second)
}

func NewRateLimiter(burstCapacity int, burstDuration time.Duration) (<-chan int, io.Closer) {
	c := make(chan int, burstCapacity)
	closer := &channelCloser{c: c}
	refill := func() {
		burstSize := burstCapacity - len(c)

		closer.mu.Lock()
		defer closer.mu.Unlock()
		if closer.closed {
			return
		}

		for range iter.N(burstSize) {
			c <- 0
		}
	}

	go func() {
		t := time.NewTicker(burstDuration)
		defer t.Stop()

		for {
			<-t.C
			if closer.Closed() {
				// channel has been closed
				break
			}

			refill()
		}
	}()

	refill()
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
	if gs.conf.IdentifyRateLimiter == nil {
		channel, closer := NewIdentifyRateLimiter()
		gs.closers = append(gs.closers, closer)

		gs.conf.IdentifyRateLimiter = &channelTaker{c: channel}
	}

	return gs
}

type GatewayStateConfig struct {
	BotToken             string
	ShardID              ShardID
	TotalNumberOfShards  uint
	Properties           GatewayIdentifyProperties
	CommandRateLimitChan <-chan int
	IdentifyRateLimiter  IdentifyRateLimiter
	GuildEvents          []event.Type
	DMEvents             []event.Type
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

func (gs *GatewayState) Write(client io.Writer, opc command.Type, payload json.RawMessage) (err error) {
	if opc != command.Heartbeat {
		// heartbeat should always be sent, regardless!
		<-gs.conf.CommandRateLimitChan
	}
	if opc == command.Identify {
		if available := gs.conf.IdentifyRateLimiter.Take(gs.conf.ShardID); !available {
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
		ready := GatewayReady{}
		if err := json.Unmarshal(payload.Data, &ready); err != nil || ready.SessionID == "" {
			return payload, length, fmt.Errorf("failed to extract session id from ready event. %w", err)
		}
		gs.sessionID = ready.SessionID
	}
	return payload, length, nil
}

func (gs *GatewayState) Process(pipe io.Reader, textWriter, closeWriter io.Writer) (payload *GatewayPayload, redundant bool, err error) {
	payload, _, err = gs.Read(pipe)
	if errors.Is(err, ErrSequenceNumberSkipped) {
		_ = gs.WriteRestartClose(closeWriter)
		return nil, true, err
	}
	if err != nil {
		return nil, false, err
	}

	redundant, err = gs.DemultiplexEvent(payload, textWriter, closeWriter)
	return payload, redundant, err
}

// Heartbeat Close method may be used if Write fails
func (gs *GatewayState) Heartbeat(client io.Writer) error {
	seq := gs.SequenceNumber()
	seqStr := strconv.FormatInt(seq, 10)
	return gs.Write(client, command.Heartbeat, []byte(seqStr))
}

// Identify Close method may be used if Write fails
func (gs *GatewayState) Identify(client io.Writer) error {
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
	resumePacket := &GatewayResume{
		BotToken:       gs.conf.BotToken,
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

func (gs *GatewayState) DemultiplexEvent(payload *GatewayPayload, textWriter, closeWriter io.Writer) (redundant bool, err error) {
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
		if _, whitelisted := gs.whitelist[payload.EventName]; !whitelisted {
			return true, nil
		}
	case opcode.InvalidSession:
		gs.InvalidateSession(closeWriter)
	case opcode.Reconnect:
		_ = gs.WriteRestartClose(closeWriter)
	default:
		// TODO: log new unhandled operation code
	}

	return false, nil
}

// DemultiplexCloseCode process close code sent by discord
func (gs *GatewayState) DemultiplexCloseCode(code closecode.Type, reason string, closeWriter io.Writer) error {
	switch code {
	case closecode.ClientReconnecting, closecode.UnknownError:
		// allow resume
	default:
		gs.InvalidateSession(closeWriter)
	}
	return &CloseError{Code: code, Reason: reason}
}
