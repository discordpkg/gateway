package gateway

import (
	"fmt"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
	"io"
	"strconv"
)

// ConnectedState handles any discord events after a successful gateway connection. The only possible state after
// this is the ClosedState or it's derivatives.
type ConnectedState struct {
	*StateCtx
}

func (st *ConnectedState) Process(payload *Payload, pipe io.Writer) error {
	switch payload.Op {
	case opcode.Heartbeat:
		seqStr := strconv.FormatInt(payload.Seq, 10)
		if err := st.StateCtx.Write(pipe, command.Heartbeat, []byte(seqStr)); err != nil {
			st.StateCtx.SetState(&ClosedState{})
			return fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
		}
	case opcode.HeartbeatACK:
		st.StateCtx.heartbeatACK.CompareAndSwap(false, true)
	case opcode.InvalidSession:
		var d bool
		if err := json.Unmarshal(payload.Data, &d); err != nil || !d {
			st.StateCtx.SetState(&ClosedState{})
		} else {
			st.StateCtx.SetState(&ResumableClosedState{st.StateCtx})
		}
		return &DiscordError{
			OpCode: payload.Op,
		}
	case opcode.Reconnect:
		st.StateCtx.SetState(&ResumableClosedState{st.StateCtx})
		return &DiscordError{
			OpCode: payload.Op,
		}
	}

	return nil
}
