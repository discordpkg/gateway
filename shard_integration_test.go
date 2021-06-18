package discordgateway

import (
	"context"
	"errors"
	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/opcode"
	"os"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/event"
)

func TestShardIntents(t *testing.T) {
	shard, err := NewShard(nil, &ShardConfig{
		BotToken: "sdjkfhsdf",
		GuildEvents: []event.Type{
			event.MessageCreate,
		},
		TotalNumberOfShards: 1,
		IdentifyProperties: GatewayIdentifyProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		},
	})
	if err != nil {
		t.Fatal("failed to create shard", err)
	}

	if shard.State.intents != intent.GuildMessages {
		t.Fatal("incorrect message intents")
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
	recordEvent := func(evt event.Type, _ []byte) {
		recordedEvents[evt] = struct{}{}
	}

	shard, err := NewShard(recordEvent, &ShardConfig{
		BotToken:            token,
		GuildEvents:         event.All(),
		DMEvents:            event.All(),
		TotalNumberOfShards: 1,
		IdentifyProperties: GatewayIdentifyProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		},
	})
	if err != nil {
		t.Fatal("failed to create shard", err)
	}

	conn, err := shard.Dial(ctx, "wss://gateway.discord.gg/?v=9&encoding=json")
	if err != nil {
		t.Fatal("failed to dial")
	}

	op, err := shard.EventLoop(ctx, conn)
	if err != nil && !(errors.Is(err, context.Canceled) || errors.Is(err, NormalCloseErr)) {
		t.Errorf("expected error to be context cancellation / normal close. Got %s", err.Error())
	}
	if op != opcode.Invalid {
		t.Errorf("expected op code to be invalid, got %d", op)
	}
}
