package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/encoding"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/event/opcode"
)

var ErrRateLimited = errors.New("unable to send message to Discord due to hitting rate limited")
var ErrIdentifyRateLimited = fmt.Errorf("can't send identify command: %w", ErrRateLimited)

type State interface {
	fmt.Stringer
	Process(payload *Payload, pipe io.Writer) error
}

// StateCloser any state implementing a Close method may overwrite the default behavior of StateCtx.Close
type StateCloser interface {
	State
	Close(closeWriter io.Writer) error
}

type StateCtx struct {
	heartbeatACK   atomic.Bool
	sequenceNumber atomic.Int64

	closed atomic.Bool
	client *Client

	SessionID        string
	ResumeGatewayURL string

	state  State
	logger Logger
}

func (ctx *StateCtx) String() string {
	return fmt.Sprintf("state-ctx(%s)", ctx.state.String())
}

func (ctx *StateCtx) SetState(state State) {
	ctx.logger.Debug("state update: %s", state)

	switch state.(type) {
	case *ClosedState:
		ctx.closed.Store(true)
	case *StateCtx:
		ctx.logger.Panic("StateCtx can not be an internal state")
	}

	ctx.state = state
}

func (ctx *StateCtx) CloseCodeHandler(payload *Payload) error {
	if payload.CloseCode == 0 {
		return nil
	}

	ctx.logger.Debug("handling close code")
	if closecode.CanReconnectAfter(payload.CloseCode) {
		ctx.SetState(&ResumableClosedState{ctx})
	} else {
		ctx.SetState(&ClosedState{})
	}

	return &DiscordError{
		CloseCode: payload.CloseCode,
		Reason:    strings.Trim(string(payload.Data), "\""),
	}
}

func (ctx *StateCtx) SessionIssueHandler(payload *Payload) error {
	switch payload.Op {
	case opcode.InvalidSession:
		var d bool
		if err := encoding.Unmarshal(payload.Data, &d); err != nil || !d {
			ctx.SetState(&ClosedState{})
		} else {
			ctx.SetState(&ResumableClosedState{ctx})
		}
	case opcode.Reconnect:
		ctx.SetState(&ResumableClosedState{ctx})
	default:
		return nil
	}

	ctx.logger.Debug("found issue with session")
	return &DiscordError{
		OpCode: payload.Op,
	}
}

func (ctx *StateCtx) Process(payload *Payload, pipe io.Writer) error {
	if err := ctx.CloseCodeHandler(payload); err != nil {
		return err
	}
	if err := ctx.SessionIssueHandler(payload); err != nil {
		return err
	}

	return ctx.state.Process(payload, pipe)
}

func (ctx *StateCtx) Close(closeWriter io.Writer) error {
	if closer, ok := ctx.state.(StateCloser); ok {
		return closer.Close(closeWriter)
	}

	// if resume details exist we close with an intent of resuming
	if ctx.SessionID != "" && ctx.ResumeGatewayURL != "" && ctx.sequenceNumber.Load() > 0 {
		return ctx.WriteRestartClose(closeWriter)
	}
	return ctx.WriteNormalClose(closeWriter)
}

func (ctx *StateCtx) Write(pipe io.Writer, evt event.Type, payload encoding.RawMessage) error {
	opc := evt.OpCode()

	ctx.logger.Debug("writing '%s' payload: %s", evt, string(payload))

	// heartbeat should always be sent.
	// Try reserving some calls for heartbeats when you configure your rate limiter.
	switch opc {
	case opcode.Dispatch, opcode.Invalid:
		return errors.New("can not send event type to Discord, it's receive only")
	case opcode.Heartbeat:
		if ok, timeout := ctx.client.commandRateLimiter.Try(ctx.client.id); !ok {
			<-time.After(timeout)
		}
	case opcode.Identify:
		if available, _ := ctx.client.identifyRateLimiter.Try(ctx.client.id); !available {
			return ErrIdentifyRateLimited
		}
	}

	packet := Payload{
		Op:   opc,
		Data: payload,
	}

	data, err := encoding.Marshal(&packet)
	if err != nil {
		return fmt.Errorf("unable to marshal packet; %w", err)
	}

	_, err = pipe.Write(data)
	return err
}

func (ctx *StateCtx) WriteNormalClose(pipe io.Writer) error {
	ctx.SetState(&ClosedState{})
	return ctx.writeClose(pipe, closecode.Normal)
}

func (ctx *StateCtx) WriteRestartClose(pipe io.Writer) error {
	ctx.SetState(&ResumableClosedState{ctx})
	return ctx.writeClose(pipe, closecode.Restarting)
}

func (ctx *StateCtx) writeClose(pipe io.Writer, code closecode.Type) error {
	writeIfOpen := func() error {
		if ctx.closed.CompareAndSwap(false, true) {
			closeCodeBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(closeCodeBuf, uint16(code))

			_, err := pipe.Write(closeCodeBuf)
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
