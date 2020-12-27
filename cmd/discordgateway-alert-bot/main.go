package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/andersfylling/disgord"
	"github.com/gobwas/ws"
	"github.com/sirupsen/logrus"

	"github.com/andersfylling/discordgateway"
	discordgatewaygobwas "github.com/andersfylling/discordgateway/cmd/gobwas"
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
	
	token := os.Getenv(EnvDiscordToken)
	if token == "" {
		logrus.Fatalf("Environment variable '%s' was not set", EnvDiscordToken)
	}
	client := disgord.New(disgord.Config{
		BotToken: token,
	})
	logrus.Info("Disgord config valid")
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
	logrus.AddHook(hook)
	
	listen(token)
}

func listen(token string) {
	logrus.Warn("STARTED")

	path := "wss://gateway.discord.gg/?v=8&encoding=json"

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		DisableTimestamp:          false,
		FullTimestamp:             true,
		TimestampFormat:           "",
	})

	shard := discordgateway.NewShard(&discordgateway.ClientStateConfig{
		Token: token,
		Intents: 0b111111111111111, // everything
		TotalNumberOfShards: 1,
		Properties:discordgateway.GatewayIdentifyProperties{
			OS:      "linux",
			Browser: "github.com/andersfylling/discordgateway v0",
			Device:  "tester",
		},
	})

reconnect:
	conn, reader, _, err := ws.Dial(context.Background(), path)
	if err != nil {
		logrus.Fatalf("failed to open websocket connection. %w", err)
	}

	if reader != nil {
		if reader.Size() > 0 {
			logrus.Error("discord sent data too quickly")
			return
		}
		ws.PutReader(reader)
	}
	

	var opcode int
	if opcode, err = discordgatewaygobwas.EventLoop(conn, shard); err != nil {
		var discordErr *discordgateway.CloseError
		if errors.As(err, &discordErr) {
			logrus.Infof("event loop exited with close code: %d", discordErr.Code)
			switch discordErr.Code {
			case 1001, 4000:
				logrus.Debug("creating resume client")
				shard = discordgateway.NewResumableShard(shard)
				goto reconnect
			case 4007, 4009:
				logrus.Debug("forcing new identify")
				shard = discordgateway.NewShardFromPrevious(shard)
				goto reconnect
			case 4001, 4002, 4003, 4004, 4005, 4008, 4010, 4011, 4012, 4013, 4014:
			default:
				logrus.Errorf("unhandled close error, with discord op code(%d): %d", opcode, discordErr.Code)
			}
		}
		logrus.Error("event loop stopped: ", err)
	} else {
		switch opcode {
		case 7:
			logrus.Debug("creating resume client, got op 7")
			shard = discordgateway.NewResumableShard(shard)
			goto reconnect
		case 9:
			logrus.Debug("creating new client, got op 9")
			shard = discordgateway.NewShardFromPrevious(shard)
			goto reconnect
		default:
			logrus.Error("shutting down without a opcode or error")
		}
	}
	logrus.Warn("STOPPED")
}

