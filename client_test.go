package gateway

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/closecode"
	"strings"
	"testing"
	"time"
)

type NoopRateLimiter struct {
}

func (rl *NoopRateLimiter) Try(_ ShardID) (bool, time.Duration) {
	return true, 0
}

type NopHeartbeatHandler struct{}

func (p *NopHeartbeatHandler) Configure(_ *StateCtx, _ time.Duration) {}

func (p *NopHeartbeatHandler) Run() {}

var commonOptions = []Option{
	WithBotToken("token"),
	WithCommandRateLimiter(&NoopRateLimiter{}),
	WithIdentifyRateLimiter(&NoopRateLimiter{}),
	WithHeartbeatHandler(&NopHeartbeatHandler{}),
}

func NewClientMust(t *testing.T, options ...Option) *Client {
	client, err := NewClient(options...)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func TestCloseFrameHandling(t *testing.T) {
	// The client supports processing the close code found in a websocket close frame. You can do this by creating a
	// payload json with the close code and reason specified
	options := append(commonOptions, []Option{}...)

	description := "You sent more than one identify payload. Don't do that!"
	data := fmt.Sprintf("{\"closecode\":%d,\"d\":\"%s\"}", closecode.AlreadyAuthenticated, description)

	client := NewClientMust(t, options...)
	client.ctx.SetState(&ConnectedState{client.ctx})

	reader := strings.NewReader(data)
	buffer := &bytes.Buffer{}

	_, err := client.ProcessNext(reader, buffer)
	if err == nil {
		t.Fatal("missing error")
	}

	if got := buffer.String(); got != "" {
		t.Error("client unexpectedly wrote to connection")
	}

	var discordErr *DiscordError
	if !errors.As(err, &discordErr) {
		t.Fatal("expected DiscordError type")
	}

	if discordErr.CloseCode != closecode.AlreadyAuthenticated {
		t.Error("wrong close code")
	}
	if discordErr.Reason != description {
		t.Errorf("wrong description. Got '%s', wants '%s'", discordErr.Reason, description)
	}
}

func TestCloseFrameTransitions(t *testing.T) {
	// The client supports processing the close code found in a websocket close frame. You can do this by creating a
	// payload json with the close code and reason specified
	options := append(commonOptions, []Option{}...)

	description := "description"
	resumeFrame := fmt.Sprintf("{\"closecode\":%d,\"d\":\"%s\"}", closecode.AlreadyAuthenticated, description)
	closeFrame := fmt.Sprintf("{\"closecode\":%d,\"data\":\"%s\"}", closecode.ShardingRequired, description)

	t.Run("close", func(t *testing.T) {
		client, err := NewClient(options...)
		if err != nil {
			t.Fatal(err)
		}

		client.ctx.SetState(&ConnectedState{client.ctx})

		reader := strings.NewReader(closeFrame)
		buffer := &bytes.Buffer{}

		_, err = client.ProcessNext(reader, buffer)
		if err == nil {
			t.Fatal("missing error")
		}

		if _, ok := client.ctx.state.(*ClosedState); !ok {
			t.Error("expected state to be closed")
		}
	})

	t.Run("resume", func(t *testing.T) {
		client, err := NewClient(options...)
		if err != nil {
			t.Fatal(err)
		}

		client.ctx.SetState(&ConnectedState{client.ctx})

		reader := strings.NewReader(resumeFrame)
		buffer := &bytes.Buffer{}

		_, err = client.ProcessNext(reader, buffer)
		if err == nil {
			t.Fatal("missing error")
		}

		if _, ok := client.ctx.state.(*ResumableClosedState); !ok {
			t.Error("expected state to be resumable")
		}
	})
}
