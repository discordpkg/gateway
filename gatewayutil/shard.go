package gatewayutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/discordpkg/gateway"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/gatewayutil/log"

	"github.com/discordpkg/gateway/intent"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/atomic"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
)

type WebsocketError struct {
	Err error
}

func (e *WebsocketError) Error() string {
	return fmt.Errorf("websocket logic failed: %w", e.Err).Error()
}

func (e *WebsocketError) Unwrap() error {
	return e.Err
}

type FrameError struct {
	Unwanted bool
	Err      error
}

func (e *FrameError) Error() string {
	return e.Err.Error()
}

type ShardConfig struct {
	BotToken string

	ShardID             uint
	TotalNumberOfShards uint

	IdentifyProperties gateway.IdentifyConnectionProperties

	GuildEvents []event.Type
	DMEvents    []event.Type

	CommandRateLimitChan <-chan int
	IdentifyRateLimiter  gateway.IdentifyRateLimiter

	// Intents does not have to be specified as these are derived from GuildEvents
	// and DMEvents. However, you can specify intents that will be merged with the derived intents.
	Intents intent.Type
}

func NewShard(shardID gateway.ShardID, botToken string, handler gateway.Handler, options ...gateway.Option) (*Shard, error) {
	state, err := gateway.NewState(botToken, options...)
	if err != nil {
		return nil, err
	}

	shard := &Shard{
		shardID:  shardID,
		options:  append(options, gateway.WithShardID(shardID)),
		botToken: botToken,
		State:    state,
		handler:  handler,
	}

	log.Debug(shard.State.String())

	return shard, nil
}

type ioWriteFlusher struct {
	writer *wsutil.Writer
}

func (i *ioWriteFlusher) Write(p []byte) (n int, err error) {
	if n, err = i.writer.Write(p); err != nil {
		return n, err
	}
	return n, i.writer.Flush()
}

type Shard struct {
	options  []gateway.Option
	botToken string

	Conn        net.Conn
	State       *gateway.State
	handler     gateway.Handler
	textWriter  io.Writer
	closeWriter io.Writer
	shardID     gateway.ShardID
}

// Dial sets up the websocket connection before identifying with the gateway.
// The url must be complete and specify api version and encoding:
//
//	"wss://gateway.discord.gg/"                     => invalid
//	"wss://gateway.discord.gg/?v=10"                 => invalid
//	"wss://gateway.discord.gg/?v=10&encoding=json"   => valid
func (s *Shard) Dial(ctx context.Context, URLString string) (connection net.Conn, err error) {
	URLString, err = ValidateDialURL(URLString)
	if err != nil {
		return nil, err
	}

	conn, reader, _, err := ws.Dial(ctx, URLString)
	if err != nil {
		return nil, err
	}

	if reader != nil {
		defer ws.PutReader(reader)
		if reader.Size() > 0 {
			_ = conn.Close()
			return nil, fmt.Errorf("unable to handle data sent ... ")
			// TODO: this should be handled somehow??
		}
	}

	s.Conn = conn
	s.textWriter = s.writer(ws.OpText)
	s.closeWriter = s.writer(ws.OpClose)
	return conn, nil
}

func (s *Shard) Write(op command.Type, data []byte) error {
	return s.State.Write(s.textWriter, op, data)
}

// Close closes the gatewayutil connection, session can not be resumed.
func (s *Shard) Close() error {
	if s.State.Closed() {
		return net.ErrClosed
	}

	_ = s.State.WriteClose(s.closeWriter, gateway.NormalCloseCode)
	_ = s.Conn.Close()
	return nil
}

// CloseWithReconnectIntent closes the gatewayutil connection, but allows the session to be resumed later on.
func (s *Shard) CloseWithReconnectIntent() error {
	if s.State.Closed() {
		return net.ErrClosed
	}

	_ = s.State.WriteClose(s.closeWriter, gateway.RestartCloseCode)
	_ = s.Conn.Close()
	return nil
}

func (s *Shard) writer(op ws.OpCode) io.Writer {
	return &ioWriteFlusher{wsutil.NewWriter(s.Conn, ws.StateClientSide, op)}
}

func (s *Shard) EventLoop(ctx context.Context) error {
	defer func() {
		if !s.State.Closed() {
			_ = s.State.WriteClose(s.closeWriter, gateway.RestartCloseCode)
			_ = s.Conn.Close()
		}
	}()

	return s.eventLoop(ctx)
}

func (s *Shard) PrepareForReconnect() error {
	options := s.options
	if s.State.HaveSessionID() {
		// setup a resume attempt
		options = append(options, gateway.WithSequenceNumber(s.State.SequenceNumber()))
		options = append(options, gateway.WithSessionID(s.State.SessionID()))
	}

	var err error
	s.State, err = gateway.NewState(s.botToken, options...)
	return err
}

func (s *Shard) eventLoop(ctx context.Context) error {
	life, kill := context.WithCancel(context.Background())
	defer func() {
		kill()
	}()

	controlHandler := wsutil.ControlFrameHandler(s.Conn, ws.StateClientSide)
	rd := wsutil.Reader{
		Source:          s.Conn,
		State:           ws.StateClientSide,
		CheckUTF8:       true,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}
	forcedReadTimeout := atomic.Bool{}
	pulser := &heart{}
	for {
		hdr, err := rd.NextFrame()
		if err != nil {
			_ = s.Conn.Close()
			// check for the "historical" i/o timeout message
			// net.go@timeoutErr # ~583 at 2020-12-25
			const ioTimeoutMessage = "i/o timeout"
			errMsg := err.Error()
			if strings.Contains(errMsg, ioTimeoutMessage) && forcedReadTimeout.Load() {
				return &WebsocketError{Err: net.ErrClosed}
			} else {
				closedConnection := strings.Contains(errMsg, "use of closed network connection")
				closedConnection = closedConnection || strings.Contains(errMsg, "use of closed connection")
				if closedConnection || errors.Is(err, io.EOF) {
					return &WebsocketError{Err: net.ErrClosed}
				} else {
					return &WebsocketError{Err: err}
				}
			}
		}
		if hdr.OpCode.IsControl() {
			// discord does send close frames so these must be handled
			if err := controlHandler(hdr, &rd); err != nil {
				var errClose wsutil.ClosedError
				if errors.As(err, &errClose) {
					if forcedReadTimeout.Load() {
						_ = s.Conn.Close()
						return &WebsocketError{Err: net.ErrClosed}
					}

					return HandleError(s.State, &gateway.WebsocketClosedError{
						Code:   uint16(errClose.Code),
						Reason: errClose.Reason,
					}, s.closeWriter)
				} else {
					return &WebsocketError{Err: err}
				}
			}
			continue
		}
		if hdr.OpCode&ws.OpText == 0 {
			// discord only uses text, even for heartbeats / ping/pong frames
			if err := rd.Discard(); err != nil {
				return &WebsocketError{Err: err}
			}
			continue
		}

		payload, _, err := s.State.Read(&rd)
		if err != nil {
			return HandleError(s.State, err, s.closeWriter)
		}

		if err = s.State.Update(payload, s.textWriter); err != nil {
			return HandleError(s.State, err, s.closeWriter)
		}

		// update our heartbeat handler & dispatch events
		switch payload.Op {
		case opcode.Dispatch:
			//
		case opcode.Hello:
			var hello *gateway.Hello
			if err := json.Unmarshal(payload.Data, &hello); err != nil {
				return fmt.Errorf("failed to extract heartbeat from hello message. %w", err)
			}

			pulser.gotAck.Store(true)
			pulser.interval = time.Duration(hello.HeartbeatIntervalMilli) * time.Millisecond
			pulser.conn = s.Conn
			pulser.shard = s.State
			pulser.forcedReadTimeout = &forcedReadTimeout

			go pulser.pulser(ctx, life, s.textWriter)
		default:
			pulser.update(payload)
		}
	}
}

type heart struct {
	interval          time.Duration
	conn              net.Conn
	shard             *gateway.State
	forcedReadTimeout *atomic.Bool
	gotAck            atomic.Bool
}

func (h *heart) update(payload *gateway.Payload) {
	switch payload.Op {
	case opcode.HeartbeatACK:
		h.gotAck.Store(true)
	}
}

func (h *heart) pulser(ctx context.Context, eventLoopCtx context.Context, writer io.Writer) {
	// gatewayutil <-> pulser
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	log.Debug(fmt.Sprintf("created heartbeat ticker with interval %s", h.interval))
loop:
	select {
	case <-eventLoopCtx.Done():
		return
	case <-ctx.Done():
		select {
		case <-eventLoopCtx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	case <-ticker.C:
		if h.gotAck.CAS(true, false) {
			if err := h.shard.Heartbeat(writer); err != nil {
				log.Error(fmt.Errorf("failed to send heartbeat. %w", err))
			} else {
				log.Debug("sent heartbeat")
				goto loop // go back to start
			}
		} else {
			log.Info("have not received heartbeat, shutting down")
		}
	}
	if h.shard.Closed() {
		// it was closed by the main go routine for this gatewayutil
		// so it should not be handing on read anymore
		return
	}

	plannedTimeoutWindow := 5 * time.Second

	log.Info("gatewayutil sent close frame: ", "heart beat failure")
	closeWriter := &ioWriteFlusher{wsutil.NewWriter(h.conn, ws.StateClientSide, ws.OpClose)}
	if err := h.shard.WriteClose(closeWriter, gateway.RestartCloseCode); err != nil && !errors.Is(err, net.ErrClosed) {
		// if the connection is already closed, it's not a big deal that we can't write the close code
		plannedTimeoutWindow = 100 * time.Millisecond
	}

	// handle network connection loss
	log.Debug("started fallback for connection issues")
	select {
	case <-time.After(plannedTimeoutWindow):
	case <-eventLoopCtx.Done():
		return
	}

	h.forcedReadTimeout.Store(true)
	log.Info("setting read deadline")
	if err := h.conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		log.Error(fmt.Errorf("failed to set read deadline. %w", err))
	}
}
