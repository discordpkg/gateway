package gateway

import (
	"bytes"
	"github.com/discordpkg/gateway/encoding"
	"github.com/discordpkg/gateway/event/opcode"
	"strings"
	"testing"
)

func TestHelloState(t *testing.T) {
	options := append(commonOptions, []Option{}...)

	t.Run("initial state", func(t *testing.T) {
		client := NewClientMust(t, options...)

		if _, ok := client.ctx.state.(*HelloState); !ok {
			t.Fatal("not hello state")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		client := NewClientMust(t, options...)

		reader := strings.NewReader("{[7]]99{{")
		_, err := client.ProcessNext(reader, nil)
		if err == nil {
			t.Error("expected error about json issue")
		}

		if _, ok := client.ctx.state.(*ClosedState); !ok {
			t.Error("not closed state")
		}
	})

	t.Run("unexpected operation", func(t *testing.T) {
		client := NewClientMust(t, options...)

		reader := strings.NewReader(`{"op":45}`)
		_, err := client.ProcessNext(reader, nil)
		if err == nil {
			t.Error("expected to fail due to wrong op code")
		}

		if _, ok := client.ctx.state.(*ClosedState); !ok {
			t.Error("not closed state")
		}
	})

	t.Run("unexpected json payload", func(t *testing.T) {
		client := NewClientMust(t, options...)

		reader := strings.NewReader(`{"op":10,"d":{"heartbeat_interval":true}}`)
		_, err := client.ProcessNext(reader, nil)
		if err == nil {
			t.Error("unmarshal should have failed")
		}

		if _, ok := client.ctx.state.(*ClosedState); !ok {
			t.Error("not closed state")
		}
	})

	t.Run("ok", func(t *testing.T) {
		client := NewClientMust(t, options...)

		reader := strings.NewReader(`{"op":10,"d":{"heartbeat_interval":45}}`)
		buffer := &bytes.Buffer{}

		_, err := client.ProcessNext(reader, buffer)
		if err != nil {
			t.Error(err)
		}

		if _, ok := client.ctx.state.(*ReadyState); !ok {
			t.Fatal("not ready state")
		}

		var payload *Payload
		if err = encoding.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatal("didn't write valid content to discord")
		}

		if payload.Op != opcode.Identify {
			t.Error("didn't write identify op code")
		}
	})
}
