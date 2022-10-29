package gateway

import (
	"fmt"
	"strconv"
	"time"

	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
	"go.uber.org/atomic"
)

type State interface {
	Process(payload *Payload, write Write) (State, error)
}

type StateCtx struct {
	heartbeatACK   atomic.Bool
	sequenceNumber atomic.Int64

	closed atomic.Bool
}

type ClosedState struct {
}

func (st *ClosedState) Process(payload *Payload, write Write) (State, error) {
	panic("closed")
}

type ResumableClosedState struct {
}

func (st *ResumableClosedState) Process(payload *Payload, write Write) (State, error) {
	panic("closed")
}

type HelloState struct {
	*StateCtx
	Identity *Identify
}

func (st *HelloState) Process(payload *Payload, write Write) (State, error) {
	data, err := json.Marshal(st.Identity)
	if err != nil {
		return &ClosedState{}, fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	var hello Hello
	if err := json.Unmarshal(payload.Data, &hello); err != nil {
		return &ClosedState{}, err
	}

	// TODO: should heartbeat have its own writer (for both text writes and close write)
	go (&HeartbeatProcess{
		StateCtx: st.StateCtx,
		interval: time.Duration(hello.HeartbeatIntervalMilli) * time.Millisecond,
	}).Run(write)

	if err = write(command.Identify, data); err != nil {
		return &ClosedState{}, err
	}

	return &ReadyState{st.StateCtx}, nil
}

type ReadyState struct {
	*StateCtx
}

func (st *ReadyState) Process(payload *Payload, write Write) (State, error) {
	var ready Ready
	if err := json.Unmarshal(payload.Data, &ready); err != nil {
		return &ClosedState{}, err
	}

	return &ConnectedState{st.StateCtx, ready}, nil
}

type ConnectedState struct {
	*StateCtx
	Ready
}

func (st *ConnectedState) Process(payload *Payload, write Write) (State, error) {
	switch payload.Op {
	case opcode.Heartbeat:
		seqStr := strconv.FormatInt(payload.Seq, 10)
		if err := write(command.Heartbeat, []byte(seqStr)); err != nil {
			return &ClosedState{}, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.HeartbeatACK:
		st.StateCtx.heartbeatACK.CAS(false, true)
	case opcode.InvalidSession:
		return &ClosedState{}, &DiscordError{
			OpCode: payload.Op,
		}
	case opcode.Reconnect:
		return &ResumableClosedState{}, &DiscordError{
			OpCode: payload.Op,
		}
	}

	return st, nil
}
