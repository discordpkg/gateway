package discordgateway

import (
	"github.com/andersfylling/discordgateway/intent"
	"github.com/andersfylling/discordgateway/json"

	"github.com/andersfylling/discordgateway/event"
)

//go:generate go run internal/generate/events/main.go
//go:generate go run internal/generate/intents/main.go

type RawMessage = json.RawMessage

type ShardID uint16

type Handler func(ShardID, event.Type, RawMessage)

type HandlerStruct struct {
	ShardID
	Name event.Type
	Data RawMessage
}

type GatewayHello struct {
	HeartbeatIntervalMilli int64 `json:"heartbeat_interval"`
}

type GatewayReady struct {
	SessionID string `json:"session_id"`
}

type GatewayResume struct {
	BotToken       string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int64  `json:"seq"`
}

type GatewayIdentifyProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

type GatewayIdentify struct {
	BotToken       string      `json:"token"`
	Properties     interface{} `json:"properties"`
	Compress       bool        `json:"compress,omitempty"`
	LargeThreshold uint8       `json:"large_threshold,omitempty"`
	Shard          [2]uint     `json:"shard"`
	Presence       interface{} `json:"presence"`
	Intents        intent.Type `json:"intents"`
}

type IdentifyRateLimiter interface {
	Take(ShardID) bool
}
