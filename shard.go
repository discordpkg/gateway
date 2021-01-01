package discordgateway

import (
	"context"
	"errors"
	"fmt"
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

func NewShard(whitelist event.Flag, handler func(event.Flag, []byte), conf *ClientStateConfig) *Shard {
	return &Shard{
		NewGatewayClient(conf),
		whitelist,
		handler,
	}
}

type Shard struct {
	*GatewayState
	whitelist event.Flag
	handler   func(event.Flag, []byte)
}

// Dial sets up the websocket connection before identifying with the gateway.
// The url must be complete and specify api version and encoding:
//  "wss://gateway.discord.gg/"                     => invalid
//  "wss://gateway.discord.gg/?v=8"                 => invalid
//  "wss://gateway.discord.gg/?v=8&encoding=json"   => valid
func (s *Shard) Dial(ctx context.Context, u *url.URL) (connection net.Conn, err error) {
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, errors.New("url scheme was not websocket (ws nor wss)")
	}
	if v := u.Query().Get("v"); v != "8" {
		return nil, errors.New("only discord api version 8 is supported")
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

	return conn, nil
}

func (s *Shard) writeClose(conn net.Conn, reason string) error {
	log.Info("shard sent close frame: ", reason)
	closeWriter := wsutil.NewWriter(conn, ws.StateClientSide, ws.OpClose)
	if err := s.WriteClose(closeWriter); err != nil {
		return fmt.Errorf("failed to write close frame. %w", err)
	}
	return nil
}

func (s *Shard) EventLoop(ctx context.Context, conn net.Conn) (opcode.OpCode, error) {
	allowResume := atomic.Bool{}
	defer func() {
		resume := allowResume.Load()
		hasSessionID := s.HaveSessionID()
		log.Debug("cleanup: ", resume, hasSessionID)
		if resume && hasSessionID {
			s.GatewayState = &GatewayState{
				conf:      s.GatewayState.conf,
				state:     newStateWithSeqNumber(s.SequenceNumber()),
				sessionID: s.GatewayState.sessionID,
			}
		} else {
			s.GatewayState = NewGatewayClient(&s.GatewayState.conf)
		}
	}()

	life, kill := context.WithCancel(context.Background())
	defer func() {
		kill()
		// cancel()
	}()

	closeConnection := func() {
		if s.Closed() {
			return
		}
		if err := s.writeClose(conn, "program shutdown"); err != nil {
			log.Fatal("failed to close connection properly: ", err)
		}
		_ = conn.Close()
	}
	defer closeConnection()

	writer := wsutil.NewWriter(conn, ws.StateClientSide, ws.OpText)
	controlHandler := wsutil.ControlFrameHandler(conn, ws.StateClientSide)
	rd := wsutil.Reader{
		Source:          conn,
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
			// check for the "historical" i/o timeout message
			// net.go@timeoutErr # ~583 at 2020-12-25
			const ioTimeoutMessage = "i/o timeout"
			if strings.Contains(err.Error(), ioTimeoutMessage) && forcedReadTimeout.Load() {
				_ = conn.Close()
				return opcode.Invalid, fmt.Errorf("closed connection due to timeout. %w", err)
			} else {
				_ = conn.Close()
				return opcode.Invalid, fmt.Errorf("failed to load next frame. %w", err)
			}
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				var normalClose wsutil.ClosedError
				if errors.As(err, &normalClose) {
					if forcedReadTimeout.Load() {
						_ = conn.Close()
					}
					switch normalClose.Code {
					case 1001, 4000:
						allowResume.Store(true)
					}
					return opcode.Invalid, &CloseError{Code: uint(normalClose.Code), Reason: normalClose.Reason}
				} else {
					return opcode.Invalid, fmt.Errorf("failed to handle control frame. %w", err)
				}
			}
			continue
		}
		if hdr.OpCode&ws.OpText == 0 {
			if err := rd.Discard(); err != nil {
				return opcode.Invalid, fmt.Errorf("failed to discard unwanted frame. %w", err)
			}
			log.Debug(fmt.Sprintf("discarded websocket frame due to wrong op: %s", string(hdr.OpCode)))
			continue
		}

		payload, length, err := s.Read(&rd)
		if err != nil {
			return opcode.Invalid, fmt.Errorf("unable to call shard read successfully: %w", err)
		}
		if length == 0 {
			return opcode.Invalid, errors.New("no data was actually read. Byte slice payload had a length of 0")
		}

		switch payload.Op {
		case opcode.EventDispatch:
			if s.handler != nil {
				s.handler(payload.EventFlag, payload.Data)
			}
		case opcode.EventHeartbeat:
			if err := s.Heartbeat(writer); err != nil {
				return payload.Op, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
			}
		case opcode.EventReconnect:
			allowResume.Store(true)
			return payload.Op, nil
		case opcode.EventInvalidSession:
			return payload.Op, nil
		case opcode.EventHello:
			if s.HaveIdentified() {
				continue
			}
			if s.HaveSessionID() {
				if err := s.Resume(writer); err != nil {
					return payload.Op, fmt.Errorf("sending resume failed. closing. %w", err)
				}
			} else {
				if err := s.Identify(writer); err != nil {
					return payload.Op, fmt.Errorf("identify failed. closing. %w", err)
				}
			}
			var hello *GatewayHello
			if err := json.Unmarshal(payload.Data, &hello); err != nil {
				return payload.Op, fmt.Errorf("failed to extract heartbeat from hello message. %w", err)
			}

			pulser.gotAck.Store(true)
			pulser.interval = time.Duration(hello.HeartbeatIntervalMilli) * time.Millisecond
			pulser.conn = conn
			pulser.shard = s
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
	shard             *Shard
	forcedReadTimeout *atomic.Bool
	gotAck            atomic.Bool
}

func (h *heart) pulser(ctx context.Context, eventLoopCtx context.Context, writer *wsutil.Writer) {
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
	if err := h.shard.writeClose(h.conn, "heart beat failure"); err != nil {
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
