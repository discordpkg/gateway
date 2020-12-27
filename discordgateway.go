package discordgateway

import (
	"encoding/json"

	"github.com/andersfylling/discordgateway/event"
)

//go:generate go run internal/generate/events/main.go
//go:generate go run internal/generate/intents/main.go

type RawMessage = json.RawMessage

type ShardID uint16

type Handler func(ShardID, event.Flag, RawMessage)
