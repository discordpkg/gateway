package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
	"io"
	"time"
)

type Hello struct {
	HeartbeatIntervalMilli int64 `json:"heartbeat_interval"`
}

type HelloState struct {
	*StateCtx
	Identity *Identify
}

func (st *HelloState) String() string {
	return "hello"
}

func (st *HelloState) Process(payload *Payload, pipe io.Writer) error {
	data, err := json.Marshal(st.Identity)
	if err != nil {
		st.StateCtx.SetState(&ClosedState{})
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if payload.Op != opcode.Hello {
		return errors.New(fmt.Sprintf("incorrect opcode: %d", int(payload.Op)))
	}

	var hello Hello
	if err := json.Unmarshal(payload.Data, &hello); err != nil {
		st.StateCtx.SetState(&ClosedState{})
		return err
	}

	var handler HeartbeatHandler
	handler, st.StateCtx.client.heartbeatHandler = st.StateCtx.client.heartbeatHandler, nil
	handler.Configure(st.StateCtx, time.Duration(hello.HeartbeatIntervalMilli)*time.Millisecond)
	go handler.Run()

	if err = st.StateCtx.Write(pipe, command.Identify, data); err != nil {
		st.StateCtx.SetState(&ClosedState{})
		return err
	}

	st.StateCtx.SetState(&ReadyState{StateCtx: st.StateCtx})
	return nil
}
