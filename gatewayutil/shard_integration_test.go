package gatewayutil

import (
	"context"
	"errors"
	"github.com/discordpkg/gateway"
	"os"
	"testing"
	"time"

	"github.com/discordpkg/gateway/event"
)

func TestShardIntents(t *testing.T) {
	shard, err := NewShard(0, "adas", nil,
		gateway.WithGuildEvents(event.MessageCreate),
		gateway.WithIdentifyConnectionProperties(&gateway.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "github.com/discordpkg/gateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		t.Fatal("failed to create gatewayutil", err)
	}
	if shard == nil {
		t.Fatal("gatewayutil instance is nil")
	}
}

func TestShard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		t.Fatal("missing token")
	}

	recordedEvents := make(map[event.Type]struct{})
	var recordEvent gateway.Handler = func(id gateway.ShardID, e event.Type, message gateway.RawMessage) {
		recordedEvents[e] = struct{}{}
	}

	shard, err := NewShard(0, token, recordEvent,
		gateway.WithGuildEvents(event.All()...),
		gateway.WithDirectMessageEvents(event.All()...),
		gateway.WithIdentifyConnectionProperties(&gateway.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "github.com/discordpkg/gateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		t.Fatal("failed to create gatewayutil", err)
	}

	if _, err = shard.Dial(ctx, "wss://gateway.discord.gg/?v=10&encoding=json"); err != nil {
		t.Fatal("failed to dial")
	}

	err = shard.EventLoop(ctx)
	var closeErr *gateway.DiscordError
	if errors.As(err, &closeErr) {
	} else if err != nil && !(errors.Is(err, context.Canceled)) {
		t.Errorf("expected error to be context cancellation / normal close. Got %s", err.Error())
	}
}
