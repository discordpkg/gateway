package shard

import (
	"context"
	"errors"
	"github.com/andersfylling/discordgateway"
	"os"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/opcode"

	"github.com/andersfylling/discordgateway/event"
)

func TestShardIntents(t *testing.T) {
	shard, err := NewShard(0, "adas", nil,
		discordgateway.WithGuildEvents(event.MessageCreate),
		discordgateway.WithIdentifyConnectionProperties(&discordgateway.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		t.Fatal("failed to create shard", err)
	}
	if shard == nil {
		t.Fatal("shard instance is nil")
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
	var recordEvent discordgateway.Handler = func(id discordgateway.ShardID, e event.Type, message discordgateway.RawMessage) {
		recordedEvents[e] = struct{}{}
	}

	shard, err := NewShard(0, token, recordEvent,
		discordgateway.WithGuildEvents(event.All()...),
		discordgateway.WithDirectMessageEvents(event.All()...),
		discordgateway.WithIdentifyConnectionProperties(&discordgateway.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		t.Fatal("failed to create shard", err)
	}

	if _, err = shard.Dial(ctx, "wss://gateway.discord.gg/?v=9&encoding=json"); err != nil {
		t.Fatal("failed to dial")
	}

	op, err := shard.EventLoop(ctx)
	var closeErr *discordgateway.CloseError
	if errors.As(err, &closeErr) {
	} else if err != nil && !(errors.Is(err, context.Canceled)) {
		t.Errorf("expected error to be context cancellation / normal close. Got %s", err.Error())
	}
	if op != opcode.Invalid {
		t.Errorf("expected op code to be invalid, got %d", op)
	}
}
