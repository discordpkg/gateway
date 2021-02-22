package discordgateway

import (
	"os"
	"testing"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/intent"
)

func TestShard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		t.Fatal("missing token")
	}

	var recordedEvents event.Flag
	recordEvent := func(evt event.Flag, _ []byte) {
		recordedEvents |= evt
	}

	_, err := NewShard(recordEvent, &ShardConfig{
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
}
