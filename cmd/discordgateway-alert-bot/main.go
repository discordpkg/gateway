package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/sirupsen/logrus"

	"github.com/andersfylling/discordgateway"
	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/log"
	"github.com/andersfylling/discordgateway/opcode"
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
		Content:                  fmt.Sprintf("[%s] %s", entry.Level.String(), entry.Message),
	})
	if err != nil {
		return fmt.Errorf("unable to dispatch discord message. %w", err)
	} else {
		return nil
	}
}


func main() {
	logger := logrus.New()
	log.LogInstance = logger
	
	token := os.Getenv(EnvDiscordToken)
	if token == "" {
		logrus.Fatalf("Environment variable '%s' was not set", EnvDiscordToken)
	}
	client := disgord.New(disgord.Config{
		BotToken: token,
	})
	logger.Info("Disgord config valid")
	// u, err := client.BotAuthorizeURL()
	// if err != nil {
	// 	logrus.Fatal("unable to generate authorization url: ", err)
	// } else {
	// 	_, _ = client.SendMsg(792482633438199860, fmt.Sprintf("<%s>", u.String()))
	// 	logrus.Printf("authorize: %s\n\n", u)
	// }

	hook := &errorHook{
		discordClient: client,
	}
	logger.AddHook(hook)
	
	listen(logger, token)
}

func listen(logger *logrus.Logger, token string) {
	logger.Warn("STARTED")

	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		DisableTimestamp:          false,
		FullTimestamp:             true,
		TimestampFormat:           "",
	})

	shard, err := discordgateway.NewShard(nil, &discordgateway.ShardConfig{
		BotToken: token,
		Events: event.All(),
		DMIntents: intent.DirectMessageReactions | intent.DirectMessageTyping | intent.DirectMessages,
		TotalNumberOfShards: 1,
		IdentifyProperties: discordgateway.GatewayIdentifyProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	path := "wss://gateway.discord.gg/?v=8&encoding=json"
	u, err := url.Parse(path)
	if err != nil {
		logger.Fatalf("invalid url string: %s", path)
	}

reconnect:
	conn, err := shard.Dial(context.Background(), u)
	if err != nil {
		logger.Fatalf("failed to open websocket connection. %w", err)
	}
	

	if op, err := shard.EventLoop(context.Background(), conn); err != nil {
		var discordErr *discordgateway.CloseError
		if errors.As(err, &discordErr) {
			logger.Infof("event loop exited with close code: %d", discordErr.Code)
			switch discordErr.Code {
			case 1001, 4000:
				logger.Debug("creating resume client")
				if !shard.HaveSessionID() {
					logger.Fatal("expected session id to exist")
				}
				goto reconnect
			case 4007, 4009:
				logger.Debug("forcing new identify")
				if shard.HaveSessionID() {
					logger.Fatal("expected session id to not exist")
				}
				goto reconnect
			case 4001, 4002, 4003, 4004, 4005, 4008, 4010, 4011, 4012, 4013, 4014:
			default:
				logger.Errorf("unhandled close error, with discord op code(%d): %d", op, discordErr.Code)
			}
		}
		logger.Error("event loop stopped: ", err)
	} else {
		logger.Infof("event loop exited with op code: %s", op)
		switch op {
		case opcode.EventReconnect:
			if !shard.HaveSessionID() {
				logger.Fatal("expected session id to exist")
			}
			goto reconnect
		case opcode.EventInvalidSession:
			if shard.HaveSessionID() {
				logger.Fatal("expected session id to not exist")
			}
			goto reconnect
		default:
			logger.Error("shutting down without a opcode or error")
		}
	}
	logger.Warn("STOPPED")
	<-time.After(5*time.Second)
}

