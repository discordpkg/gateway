package gateway

import (
	"io"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/event/opcode"
)

type Resume struct {
	BotToken       string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int64  `json:"seq"`
}

// ResumeState wraps a ConnectedState until a Resumed event is received from Discord...
type ResumeState struct {
	parentState *ConnectedState
}

func (st *ResumeState) String() string {
	return "resume"
}

func (st *ResumeState) Process(payload *Payload, pipe io.Writer) error {
	if err := st.parentState.Process(payload, pipe); err != nil {
		return err
	}

	if payload.Op == opcode.Dispatch && payload.EventName == event.Resumed {
		// simply unwrap the existing connected state
		st.parentState.ctx.SetState(st.parentState)
	}

	return nil
}
