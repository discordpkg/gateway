package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/event/opcode"
	"github.com/discordpkg/gateway/json"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

var ErrRateLimited = errors.New("unable to send message to Discord due to hitting rate limited")
var ErrIdentifyRateLimited = fmt.Errorf("can't send identify command: %w", ErrRateLimited)

type State interface {
	Process(payload *Payload, pipe io.Writer) error
	//Close(*StateCtx) error
}

type StateCtx struct {
	heartbeatACK   atomic.Bool
	sequenceNumber atomic.Int64

	closed atomic.Bool
	client *Client

	SessionID        string
	ResumeGatewayURL string

	state State
}

func (ctx *StateCtx) SetState(state State) {
	switch state.(type) {
	case *ClosedState:
		ctx.closed.Store(true)
	}

	ctx.state = state
}

func (ctx *StateCtx) SessionIssueHandler(payload *Payload) error {
	switch payload.Op {
	case opcode.InvalidSession:
		var d bool
		if err := json.Unmarshal(payload.Data, &d); err != nil || !d {
			ctx.SetState(&ClosedState{})
		} else {
			ctx.SetState(&ResumableClosedState{ctx})
		}
	case opcode.Reconnect:
		ctx.SetState(&ResumableClosedState{ctx})
	default:
		return nil
	}

	return &DiscordError{
		OpCode: payload.Op,
	}
}

func (ctx *StateCtx) Process(payload *Payload, pipe io.Writer) error {
	if err := ctx.SessionIssueHandler(payload); err != nil {
		return err
	}

	return ctx.state.Process(payload, pipe)
}

func (ctx *StateCtx) Write(pipe io.Writer, evt event.Type, payload json.RawMessage) error {
	opc := evt.OpCode()

	// heartbeat should always be sent.
	// Try reserving some calls for heartbeats when you configure your rate limiter.
	switch opc {
	case opcode.Dispatch:
		return errors.New("can not send event type to Discord, it's receive only")
	case opcode.Heartbeat:
		if ok, timeout := ctx.client.commandRateLimiter.Try(); !ok {
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

	data, err := json.Marshal(&packet)
	if err != nil {
		return fmt.Errorf("unable to marshal packet; %w", err)
	}

	_, err = pipe.Write(data)
	return err
}

func (ctx *StateCtx) WriteNormalClose(pipe io.Writer) error {
	return ctx.writeClose(pipe, NormalCloseCode)
}

func (ctx *StateCtx) WriteRestartClose(pipe io.Writer) error {
	ctx.SetState(&ResumableClosedState{ctx})
	return ctx.writeClose(pipe, RestartCloseCode)
}

func (ctx *StateCtx) writeClose(pipe io.Writer, code uint16) error {
	writeIfOpen := func() error {
		if ctx.closed.CompareAndSwap(false, true) {
			closeCodeBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(closeCodeBuf, code)

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
