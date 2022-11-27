package gateway

import (
	"errors"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
)

// Option for initializing a new gateway state. An option must be deterministic regardless
// of when or how many times it is executed.
type Option func(client *Client) error

func WithDirectMessageEvents(events ...event.Type) Option {
	set := util.Set[event.Type]{}
	set.Add(events...)
	deduplicated := set.ToSlice()

	return func(client *Client) error {
		if len(deduplicated) != len(events) {
			return errors.New("duplicated direct message events found")
		}
		if client.intents > 0 {
			return errors.New("'DirectMessageEvents' can not be set when using 'Intents' option")
		}

		client.directMessageEvents = events
		return nil
	}
}

func WithGuildEvents(events ...event.Type) Option {
	set := util.Set[event.Type]{}
	set.Add(events...)
	deduplicated := set.ToSlice()

	return func(client *Client) error {
		if len(deduplicated) != len(events) {
			return errors.New("duplicated guild events found")
		}
		if client.intents > 0 {
			return errors.New("'GuildEvents' can not be set when using 'Intents' option")
		}

		client.guildEvents = events
		return nil
	}
}

func WithIntents(intents intent.Type) Option {
	return func(client *Client) error {
		if len(client.directMessageEvents) > 0 || len(client.guildEvents) > 0 {
			return errors.New("'Intents' can not be used along with 'DirectMessageEvents' and/or 'GuildEvents'")
		}

		client.intents = intents
		return nil
	}
}

func WithShardID(id ShardID) Option {
	return func(client *Client) error {
		client.id = id
		return nil
	}
}

func WithShardCount(count int) Option {
	if count < 0 {
		panic("shard count must be above 0")
	}

	return func(client *Client) error {
		client.totalNumberOfShards = count
		return nil
	}
}

func WithSessionID(id string) Option {
	if id == "" {
		panic("session id is not set")
	}

	return func(client *Client) error {
		client.ctx.SessionID = id
		return nil
	}
}

func WithIdentifyConnectionProperties(properties *IdentifyConnectionProperties) Option {
	return func(client *Client) error {
		client.connectionProperties = properties
		return nil
	}
}

func WithCommandRateLimiter(ratelimiter CommandRateLimiter) Option {
	return func(client *Client) error {
		client.commandRateLimiter = ratelimiter
		return nil
	}
}

func WithIdentifyRateLimiter(ratelimiter IdentifyRateLimiter) Option {
	return func(client *Client) error {
		client.identifyRateLimiter = ratelimiter
		return nil
	}
}

// WithHeartbeatHandler allows overwriting default heartbeat behavior.
// Basic behavior is achieved with the DefaultHeartbeatHandler:
//
//	 NewClient(
//	 	WithHeartbeatHandler(&DefaultHeartbeatHandler{
//				TextWriter:
//			})
//	 )
func WithHeartbeatHandler(handler HeartbeatHandler) Option {
	return func(client *Client) error {
		client.heartbeatHandler = handler
		return nil
	}
}
