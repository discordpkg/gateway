package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

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

func (ctx *StateCtx) Process(payload *Payload, pipe io.Writer) error {
	return ctx.state.Process(payload, pipe)
}

func (ctx *StateCtx) Write(pipe io.Writer, opc command.Type, payload json.RawMessage) error {
	// heartbeat should always be sent.
	// Try reserving some calls for heartbeats when you configure your rate limiter.
	if opc != command.Heartbeat {
		if ok, timeout := ctx.client.commandRateLimiter.Try(); !ok {
			<-time.After(timeout)
		}
	}
	if opc == command.Identify {
		if available, _ := ctx.client.identifyRateLimiter.Try(ctx.client.id); !available {
			return errors.New("identify rate limiter denied shard to identify")
		}
	}

	packet := Payload{
		Op:   opcode.Type(opc),
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
