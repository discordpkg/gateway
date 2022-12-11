package gateway

import (
	"fmt"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/event/opcode"
	"io"
	"strconv"
)

// ConnectedState handles any discord events after a successful gateway connection. The only possible state after
// this is the ClosedState or it's derivatives such as a resumable state.
//
// See the Discord documentation for more information:
//   - https://discord.com/developers/docs/topics/gateway#dispatch-events
//   - https://discord.com/developers/docs/topics/gateway#heartbeat-interval-example-heartbeat-ack
//   - https://discord.com/developers/docs/topics/gateway#heartbeat-requests
type ConnectedState struct {
	*StateCtx
}

func (st *ConnectedState) Process(payload *Payload, pipe io.Writer) error {
	switch payload.Op {
	case opcode.Heartbeat:
		seqStr := strconv.FormatInt(payload.Seq, 10)
		if err := st.StateCtx.Write(pipe, event.Heartbeat, []byte(seqStr)); err != nil {
			st.StateCtx.SetState(&ClosedState{})
			return fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.HeartbeatACK:
		st.StateCtx.heartbeatACK.CompareAndSwap(false, true)
	case opcode.Dispatch:
		if st.StateCtx.client.eventHandler == nil {
			return nil
		}

		if _, ok := st.StateCtx.client.allowlist[payload.EventName]; !ok {
			return nil
		}

		st.StateCtx.client.eventHandler(st.StateCtx.client.id, payload.EventName, payload.Data)
	}

	return nil
}
