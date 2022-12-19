package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/encoding"
	"time"

	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/event/opcode"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
)

//go:generate go run internal/generate/events/main.go
//go:generate go run internal/generate/closecode/main.go

type ShardID uint

type Payload struct {
	Op        opcode.Type         `json:"op"`
	Data      encoding.RawMessage `json:"d"`
	Seq       int64               `json:"s,omitempty"`
	EventName event.Type          `json:"t,omitempty"`

	// CloseCode is a special case for this library.
	// You can specify an io.Reader which produces relevant closecode data
	// for correct handling of close frames
	CloseCode closecode.Type `json:"closecode,omitempty"`
}

func (p Payload) String() string {
	return fmt.Sprintf("{\n\t\"op\":%d,\n\t\"data\": %s\n\t\"seq\":%d\n}", p.Op, string(p.Data), p.Seq)
}

var ErrSequenceNumberSkipped = errors.New("the sequence number increased with more than 1, events lost")

type DiscordError struct {
	CloseCode closecode.Type
	OpCode    opcode.Type
	Reason    string
}

func (c *DiscordError) Error() string {
	return fmt.Sprintf("[%d | %d]: %s", c.CloseCode, c.OpCode, c.Reason)
}

func (c DiscordError) CanReconnect() bool {
	return closecode.CanReconnectAfter(c.CloseCode) || opcode.CanReconnectAfter(c.OpCode)
}

type Handler func(shardID ShardID, evt event.Type, data encoding.RawMessage)

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
	Shard          [2]int      `json:"shard"`
	Presence       interface{} `json:"presence"`
	Intents        intent.Type `json:"intents"`
}

type RateLimiter interface {
	Try(ShardID) (bool, time.Duration)
}
