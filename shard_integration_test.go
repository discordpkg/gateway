package discordgateway

import (
	"context"
	"errors"
	"github.com/andersfylling/discordgateway/opcode"
	"os"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/intent"
)

func TestShard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
	defer cancel()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		t.Fatal("missing token")
	}

	var recordedEvents event.Flag
	recordEvent := func(evt event.Flag, _ []byte) {
		recordedEvents |= evt
	}

	shard, err := NewShard(recordEvent, &ShardConfig{
		BotToken:            token,
		Events:              event.All(),
		DMIntents:           intent.DirectMessageReactions | intent.DirectMessageTyping | intent.DirectMessages,
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

	conn, err := shard.Dial(ctx, "wss://gateway.discord.gg/?v=8&encoding=json")
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
