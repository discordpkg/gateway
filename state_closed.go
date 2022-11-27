package gateway

import "io"

type ClosedState struct {
}

func (st *ClosedState) Process(payload *Payload, _ io.Writer) error {
	panic("closed")
}

type ResumableClosedState struct {
	*StateCtx
}

func (st *ResumableClosedState) Process(payload *Payload, _ io.Writer) error {
	panic("closed")
}
