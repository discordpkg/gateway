package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/gatewayshard"
	"os"
	"runtime"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/sirupsen/logrus"

	"github.com/discordpkg/gateway"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/log"
)

const EnvDiscordToken = "DISCORD_TOKEN"

type errorHook struct {
	discordClient *disgord.Client
}

var _ logrus.Hook = &errorHook{}

func (e errorHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
		logrus.WarnLevel,
	}
}

func (e errorHook) Fire(entry *logrus.Entry) error {
	_, err := e.discordClient.Channel(792482633438199860).CreateMessage(&disgord.CreateMessageParams{
		Content: fmt.Sprintf("[%s] %s", entry.Level.String(), entry.Message),
	})
	if err != nil {
		return fmt.Errorf("unable to dispatch discord message. %w", err)
	} else {
		return nil
	}
}

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "",
	})
	log.LogInstance = logger

	token := os.Getenv(EnvDiscordToken)
	if token == "" {
		logrus.Fatalf("Environment variable '%s' was not set", EnvDiscordToken)
	}
	client := disgord.New(disgord.Config{
		BotToken: token,
	})
	logger.Info("Disgord config valid")

	u, err := client.BotAuthorizeURL(0, nil)
	if err != nil {
		logrus.Fatal("unable to generate authorization url: ", err)
	} else {
		logrus.Printf("authorize: %s\n\n", u)
	}

	hook := &errorHook{
		discordClient: client,
	}
	logger.AddHook(hook)

	listen(logger, token)
}

type DiscordEvent struct {
	Topic event.Type
	Data  []byte
}

func listen(logger *logrus.Logger, token string) {
	logger.Warn("STARTED")

	shard, err := gatewayshard.NewShard(0, token, nil,
		gateway.WithGuildEvents(event.All()...),
		gateway.WithIdentifyConnectionProperties(&gateway.IdentifyConnectionProperties{
			OS:      runtime.GOOS,
			Browser: "github.com/discordpkg/gateway v0",
			Device:  "tester",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

reconnect:
	if _, err = shard.Dial(context.Background(), "wss://gateway.discord.gg/?v=9&encoding=json"); err != nil {
		logger.Fatal(fmt.Errorf("failed to open websocket connection. %w", err))
	}

	// process websocket messages as they arrive and trigger the handler whenever relevant
	if err = shard.EventLoop(context.Background()); err != nil {
		reconnect := true

		var discordErr *gateway.DiscordError
		if errors.As(err, &discordErr) {
			reconnect = discordErr.CanReconnect()
		}

		if reconnect {
			logger.Info(fmt.Errorf("reconnecting: %w", err))
			if err := shard.PrepareForReconnect(); err != nil {
				logger.Fatal("failed to prepare for reconnect:", err)
			}
			goto reconnect
		}
	}

	logger.Error("event loop stopped: ", err)
	<-time.After(5 * time.Second)
}
