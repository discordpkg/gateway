package gateway

import (
	"github.com/discordpkg/gateway/encoding"
	"testing"
)

func TestPayload(t *testing.T) {
	t.Run("unmarshal", func(t *testing.T) {
		data := []byte(`{"op":123, "d":{"test":123}, "s":234, "t":"test"}`)

		var payload *Payload
		if err := encoding.Unmarshal(data, &payload); err != nil {
			t.Error("failed to unmarshal data into payload type")
		}

		if payload.Op != 123 {
			t.Error("wrong op code")
		}
		if payload.Seq != 234 {
			t.Error("wrong sequence")
		}
		if payload.EventName != "test" {
			t.Error("wrong event name")
		}
		if string(payload.Data) != `{"test":123}` {
			t.Error("wrong data value")
		}
	})
}
