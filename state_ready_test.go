package gateway

import (
	"bytes"
	"testing"
)

func NewReadyState(t *testing.T, options ...Option) *ReadyState {
	// ensure it's properly setup via the client constructor
	client := NewClientMust(t, options...)
	client.ctx.SetState(&ReadyState{ctx: client.ctx})
	return client.ctx.state.(*ReadyState)
}

func TestReadyState_String(t *testing.T) {
	state := &ReadyState{ctx: nil}
	got := state.String()
	wants := "ready"
	if got != wants {
		t.Errorf("incorrect state name. Got %s, wants %s", got, wants)
	}
}

func TestReadyState_Process(t *testing.T) {
	options := append(commonOptions, []Option{}...)

	t.Run("unexpected payload", func(t *testing.T) {
		state := NewReadyState(t, options...)

		// try using a hello payload
		payload := &Payload{Op: 10, Data: []byte(`{"heartbeat_interval":45}`)}
		buffer := &bytes.Buffer{}

		if err := state.Process(payload, buffer); err == nil {
			t.Fatal("should have failed")
		}

		if _, ok := state.ctx.state.(*ClosedState); !ok {
			t.Error("state was not closed")
		}
	})

	t.Run("ok", func(t *testing.T) {
		state := NewReadyState(t, options...)

		// try using a hello payload
		payload := &Payload{Op: 0, Data: []byte(`{"v":10, "session_id": "test", "resume_gateway_url": "test.com"}`)}
		buffer := &bytes.Buffer{}

		if err := state.Process(payload, buffer); err != nil {
			t.Fatal("should properly handle the dispatch payload")
		}

		if _, ok := state.ctx.state.(*ConnectedState); !ok {
			t.Fatal("state was not set to connected")
		}

		if state.ctx.SessionID == "" {
			t.Error("forgot to save session id")
		}
		if state.ctx.ResumeGatewayURL == "" {
			t.Error("forgot to save resume url")
		}
	})
}
