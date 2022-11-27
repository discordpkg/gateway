package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
	"io"
)

type Ready struct {
	SessionID        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

type ReadyState struct {
	*StateCtx
}

func (st *ReadyState) Process(payload *Payload, _ io.Writer) error {
	if payload.Op != opcode.Dispatch {
		return errors.New(fmt.Sprintf("incorrect opcode: %d, wants %d", int(payload.Op), int(opcode.Dispatch)))
	}

	var ready Ready
	if err := json.Unmarshal(payload.Data, &ready); err != nil {
		st.StateCtx.SetState(&ClosedState{})
		return err
	}

	st.StateCtx.SessionID = ready.SessionID
	st.StateCtx.ResumeGatewayURL = ready.ResumeGatewayURL

	st.StateCtx.SetState(&ConnectedState{StateCtx: st.StateCtx})
	return nil
}
