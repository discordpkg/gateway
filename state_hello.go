package gateway

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/event/opcode"
	"github.com/discordpkg/gateway/json"
)

type Hello struct {
	HeartbeatIntervalMilli int64 `json:"heartbeat_interval"`
}

// HelloState is one of several initial state for the client. It's responsibility are as follows
//  1. Process incoming Hello event
//  2. Initiate a heartbeat process
//  3. Send Identify message
//  4. Transition to the ReadyState
//
// This state is responsible for handling the Hello phase of the gateway connection. See the Discord documentation
// for more information:
//   - https://discord.com/developers/docs/topics/gateway#connecting
//   - https://discord.com/developers/docs/topics/gateway#hello-event
//   - https://discord.com/developers/docs/topics/gateway#sending-heartbeats
//   - https://discord.com/developers/docs/topics/gateway#identifying
type HelloState struct {
	ctx      *StateCtx
	Identity *Identify
}

func (st *HelloState) String() string {
	return "hello"
}

func (st *HelloState) Process(payload *Payload, pipe io.Writer) error {
	data, err := json.Marshal(st.Identity)
	if err != nil {
		st.ctx.SetState(&ClosedState{})
		return fmt.Errorf("unable to marshal identify payload. %w", err)
	}

	if payload.Op != opcode.Hello {
		return errors.New(fmt.Sprintf("incorrect opcode: %d", int(payload.Op)))
	}

	var hello Hello
	if err := json.Unmarshal(payload.Data, &hello); err != nil {
		st.ctx.SetState(&ClosedState{})
		return err
	}

	st.ctx.logger.Debug("starting heartbeat process")
	var handler HeartbeatHandler
	handler, st.ctx.client.heartbeatHandler = st.ctx.client.heartbeatHandler, nil
	handler.Configure(st.ctx, time.Duration(hello.HeartbeatIntervalMilli)*time.Millisecond)
	st.ctx.heartbeatACK.Store(true)
	go handler.Run()

	if err = st.ctx.Write(pipe, event.Identify, data); err != nil {
		st.ctx.SetState(&ClosedState{})
		return err
	}

	st.ctx.SetState(&ReadyState{ctx: st.ctx})
	return nil
}
