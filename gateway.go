package gateway

import (
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/json"
	"time"

	"github.com/discordpkg/gateway/event"
)

//go:generate go run internal/generate/events/main.go
//go:generate go run internal/generate/opcode/main.go
//go:generate go run internal/generate/closecode/main.go

type RawMessage = json.RawMessage

type ShardID uint

type Handler func(ShardID, event.Type, RawMessage)

type HandlerStruct struct {
	ShardID
	Name event.Type
	Data RawMessage
}

type Hello struct {
	HeartbeatIntervalMilli int64 `json:"heartbeat_interval"`
}

type Ready struct {
	SessionID string `json:"session_id"`
}

type Resume struct {
	BotToken       string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int64  `json:"seq"`
}

type IdentifyConnectionProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type Identify struct {
	BotToken       string      `json:"token"`
	Properties     interface{} `json:"properties"`
	Compress       bool        `json:"compress,omitempty"`
	LargeThreshold uint8       `json:"large_threshold,omitempty"`
	Shard          [2]uint     `json:"shard"`
	Presence       interface{} `json:"presence"`
	Intents        intent.Type `json:"intents"`
}

type IdentifyRateLimiter interface {
	Try(ShardID) (bool, time.Duration)
}

type CommandRateLimiter interface {
	Try() (bool, time.Duration)
}
