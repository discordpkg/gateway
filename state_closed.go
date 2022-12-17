package gateway

import "io"

type ClosedState struct {
}

func (st *ClosedState) String() string {
	return "closed"
}

func (st *ClosedState) Process(payload *Payload, _ io.Writer) error {
	panic("closed")
}

func (st *ClosedState) Close(closeWriter io.Writer) error {
	panic("ClosedState")
}

type ResumableClosedState struct {
	ctx *StateCtx
}

func (st *ResumableClosedState) String() string {
	return "closed-resumable"
}

func (st *ResumableClosedState) Process(payload *Payload, _ io.Writer) error {
	panic("closed")
}

func (st *ResumableClosedState) Close(closeWriter io.Writer) error {
	panic("ResumableClosedState")
}
