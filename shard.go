package discordgateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/intent"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"
	"github.com/andersfylling/discordgateway/log"
	"github.com/andersfylling/discordgateway/opcode"
)

// ErrClosed https://tip.golang.org/pkg/net/#ErrClosed
type ErrClosed struct {
	err error
}

func (e ErrClosed) Error() string {
	return e.err.Error()
}

var UnhandledControlFrameErr = errors.New("unhandled control frame")

type ShardConfig struct {
	BotToken string

	ShardID             uint
	TotalNumberOfShards uint

	IdentifyProperties GatewayIdentifyProperties

	GuildEvents []event.Type
	DMEvents    []event.Type

	CommandRateLimitChan <-chan int
	IdentifyRateLimiter  IdentifyRateLimiter

	// Intents does not have to be specified as these are derived from GuildEvents
	// and DMEvents. However, you can specify intents and it will be merged with the derived intents.
	Intents intent.Type
}

func NewShard(handler Handler, conf *ShardConfig) (*Shard, error) {
	gatewayConf := GatewayStateConfig{
		BotToken:             conf.BotToken,
		ShardID:              ShardID(conf.ShardID),
		TotalNumberOfShards:  conf.TotalNumberOfShards,
		Properties:           conf.IdentifyProperties,
		GuildEvents:          conf.GuildEvents,
		DMEvents:             conf.DMEvents,
		CommandRateLimitChan: conf.CommandRateLimitChan,
		IdentifyRateLimiter:  conf.IdentifyRateLimiter,
	}
	shard := &Shard{
		State:   NewGatewayClient(&gatewayConf),
		handler: handler,
	}
	shard.State.intents |= conf.Intents

	whitelistToSlice := func() (events []event.Type) {
		for e := range shard.State.whitelist {
			events = append(events, e)
		}
		return events
	}

	log.Debug("intents: ", shard.State.intents)
	log.Debug("whitelisted events: ", whitelistToSlice())

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
	Conn       net.Conn
	State      *GatewayState
	handler    Handler
	textWriter io.Writer
}

// Dial sets up the websocket connection before identifying with the gateway.
// The url must be complete and specify api version and encoding:
//  "wss://gateway.discord.gg/"                     => invalid
//  "wss://gateway.discord.gg/?v=9"                 => invalid
//  "wss://gateway.discord.gg/?v=9&encoding=json"   => valid
func (s *Shard) Dial(ctx context.Context, URLString string) (connection net.Conn, err error) {
	u, urlErr := url.Parse(URLString)
	if urlErr != nil {
		return nil, err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, errors.New("url scheme was not websocket (ws nor wss)")
	}
	if v := u.Query().Get("v"); v != "9" && v != "8" {
		return nil, errors.New("only discord api version [8, 9] is supported")
	}
	if encoding := u.Query().Get("encoding"); encoding != "json" {
		return nil, errors.New("currently, only supports json encoding of discord data")
	}

	conn, reader, _, err := ws.Dial(ctx, u.String())
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
	return conn, nil
}

func (s *Shard) Write(op opcode.OpCode, data []byte) error {
	return s.State.Write(s.textWriter, op, data)
}

func (s *Shard) writer(op ws.OpCode) io.Writer {
	return &ioWriteFlusher{wsutil.NewWriter(s.Conn, ws.StateClientSide, op)}
}

func writeClose(closer func(io.Writer) error, conn net.Conn, reason string) error {
	log.Info("shard sent close frame: ", reason)
	closeWriter := &ioWriteFlusher{wsutil.NewWriter(conn, ws.StateClientSide, ws.OpClose)}
	if err := closer(closeWriter); err != nil && !errors.Is(err, net.ErrClosed) {
		// if the connection is already closed, it's not a big deal that we can't write the close code
		return fmt.Errorf("failed to write close frame. %w", err)
	}
	return nil
}

func (s *Shard) EventLoop(ctx context.Context) (opcode.OpCode, error) {
	defer func() {
		if s.State.HaveSessionID() {
			s.State = &GatewayState{
				conf:      s.State.conf,
				state:     newStateWithSeqNumber(s.State.SequenceNumber()),
				sessionID: s.State.sessionID,
			}
		} else {
			s.State = NewGatewayClient(&s.State.conf)
		}
	}()

	life, kill := context.WithCancel(context.Background())
	defer func() {
		kill()
	}()

	closeConnection := func() {
		if s.State.Closed() {
			return
		}

		if err := writeClose(s.State.WriteRestartClose, s.Conn, "program shutdown"); err != nil {
			log.Error("failed to close connection properly: ", err)
		}
		_ = s.Conn.Close()
	}
	defer closeConnection()

	writer := s.writer(ws.OpText)
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
			if strings.Contains(err.Error(), ioTimeoutMessage) && forcedReadTimeout.Load() {
				return opcode.Invalid, fmt.Errorf("closed connection due to timeout. %w", err)
			} else {
				closedConnection := strings.Contains(err.Error(), "use of closed network connection")
				closedConnection = closedConnection || strings.Contains(err.Error(), "use of closed connection")
				if closedConnection || errors.Is(err, io.EOF) {
					return opcode.Invalid, &ErrClosed{err}
				} else {
					return opcode.Invalid, fmt.Errorf("failed to load next frame. %w", err)
				}
			}
		}
		if hdr.OpCode.IsControl() {
			// discord does send close frames so these must be handled
			if err := controlHandler(hdr, &rd); err != nil {
				var normalClose wsutil.ClosedError
				if errors.As(err, &normalClose) {
					if forcedReadTimeout.Load() {
						_ = s.Conn.Close()
					}
					switch normalClose.Code {
					case 1001, 4000: // allow resume
					default:
						closeWriter := wsutil.NewWriter(s.Conn, ws.StateClientSide, ws.OpClose)
						s.State.InvalidateSession(closeWriter)
					}
					if normalClose.Code == 1000 {
						return opcode.Invalid, NormalCloseErr
					}
					return opcode.Invalid, &CloseError{Code: uint(normalClose.Code), Reason: normalClose.Reason}
				} else {
					return opcode.Invalid, fmt.Errorf("failed to handle control frame. %w", err)
				}
			}
			continue
		}
		if hdr.OpCode&ws.OpText == 0 {
			// discord only uses text, even for heartbeats / ping/pong frames
			if err := rd.Discard(); err != nil {
				return opcode.Invalid, fmt.Errorf("failed to discard unwanted frame. %w", err)
			}
			continue
		}

		payload, length, err := s.State.Read(&rd)
		if err != nil {
			return opcode.Invalid, fmt.Errorf("unable to call shard read successfully: %w", err)
		}
		if length == 0 {
			return opcode.Invalid, errors.New("no data was actually read. Byte slice payload had a length of 0")
		}

		redundant, err := s.State.DemultiplexEvent(payload, writer)
		if redundant {
			continue
		}
		if err != nil {
			return payload.Op, err
		}

		switch payload.Op {
		case opcode.EventDispatch:
			if s.handler != nil {
				s.handler(s.State.conf.ShardID, payload.EventName, payload.Data)
			}
		case opcode.EventInvalidSession:
			closeWriter := wsutil.NewWriter(s.Conn, ws.StateClientSide, ws.OpClose)
			s.State.InvalidateSession(closeWriter)
			return payload.Op, nil
		case opcode.EventHello:
			var hello *GatewayHello
			if err := json.Unmarshal(payload.Data, &hello); err != nil {
				return payload.Op, fmt.Errorf("failed to extract heartbeat from hello message. %w", err)
			}

			pulser.gotAck.Store(true)
			pulser.interval = time.Duration(hello.HeartbeatIntervalMilli) * time.Millisecond
			pulser.conn = s.Conn
			pulser.shard = s.State
			pulser.forcedReadTimeout = &forcedReadTimeout

			go pulser.pulser(ctx, life, writer)
		case opcode.EventHeartbeatACK:
			pulser.gotAck.Store(true)
		default:
		}
	}

	return opcode.Invalid, nil
}

type heart struct {
	interval          time.Duration
	conn              net.Conn
	shard             *GatewayState
	forcedReadTimeout *atomic.Bool
	gotAck            atomic.Bool
}

func (h *heart) pulser(ctx context.Context, eventLoopCtx context.Context, writer io.Writer) {
	// shard <-> pulser
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
		// it was closed by the main go routine for this shard
		// so it should not be handing on read anymore
		return
	}

	plannedTimeoutWindow := 5 * time.Second
	if err := writeClose(h.shard.WriteRestartClose, h.conn, "heart beat failure"); err != nil {
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
