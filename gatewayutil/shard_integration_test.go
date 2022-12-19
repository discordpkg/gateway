package gatewayutil

import (
	"context"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/encoding"
	"os"
	"testing"
	"time"

	"github.com/discordpkg/gateway/gatewayutil/log"

	"github.com/discordpkg/gateway"

	"github.com/discordpkg/gateway/event"
)

type printLogger struct{}

func (l *printLogger) Debug(format string, data ...interface{}) {
	fmt.Println(fmt.Sprintf("[DEBUG] "+format, data...))
}
func (l *printLogger) Info(format string, data ...interface{}) {
	fmt.Println(fmt.Sprintf("[INFO] "+format, data...))
}
func (l *printLogger) Warn(format string, data ...interface{}) {
	fmt.Println(fmt.Sprintf("[WARN] "+format, data...))
}
func (l *printLogger) Error(format string, data ...interface{}) {
	fmt.Println(fmt.Sprintf("[ERROR] "+format, data...))
}
func (l *printLogger) Panic(format string, data ...interface{}) {
	stmt := fmt.Sprint(data...)
	fmt.Println(fmt.Sprintf("[PANIC] "+format, data...))
	panic(stmt)
}

func TestShardIntents(t *testing.T) {
	shard, err := NewShard(
		gateway.WithShardInfo(0, 1),
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

	envVar := os.Getenv("DISCORD_TOKEN_ENVVAR")
	if envVar == "" {
		envVar = "DISCORD_TOKEN"
	}

	token := os.Getenv(envVar)
	if token == "" {
		t.Skip("missing bot token")
	}

	recordedEvents := make(map[event.Type]struct{})
	var recordEvent gateway.Handler = func(id gateway.ShardID, e event.Type, message encoding.RawMessage) {
		recordedEvents[e] = struct{}{}
	}

	shard, err := NewShard(
		gateway.WithLogger(&printLogger{}),
		gateway.WithBotToken(token),
		gateway.WithEventHandler(recordEvent),
		gateway.WithShardInfo(0, 1),
		gateway.WithGuildEvents(event.All()...),
		gateway.WithDirectMessageEvents(event.All()...),
		gateway.WithCommandRateLimiter(NewCommandRateLimiter()),
		gateway.WithIdentifyRateLimiter(NewLocalIdentifyRateLimiter()),
		gateway.WithIdentifyConnectionProperties(&gateway.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "github.com/discordpkg/gateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		t.Fatal("failed to create gatewayutil", err)
	}

	_, err = shard.Dial(ctx, func() (string, error) {
		return "wss://gateway.discord.gg/?v=10&encoding=json", nil
	})
	if err != nil {
		t.Fatal("failed to dial")
	}

	err = shard.EventLoop(ctx)
	var closeErr *gateway.DiscordError
	if errors.As(err, &closeErr) {
		log.Error("%s", err)
	} else if err != nil && !(errors.Is(err, context.Canceled)) {
		t.Errorf("expected error to be context cancellation / normal close. Got %s", err.Error())
	}
}
