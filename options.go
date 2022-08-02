package gateway

import (
	"errors"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
)

// Option for initializing a new gateway state. An option must be deterministic regardless
// of when or how many times it is executed.
type Option func(st *GatewayState) error

func WithDirectMessageEvents(events ...event.Type) Option {
	set := util.Set[event.Type]{}
	set.Add(events...)
	deduplicated := set.ToSlice()

	return func(st *GatewayState) error {
		if len(deduplicated) != len(events) {
			return errors.New("duplicated direct message events found")
		}
		if st.intents > 0 {
			return errors.New("'DirectMessageEvents' can not be set when using 'Intents' option")
		}

		st.directMessageEvents = events
		return nil
	}
}

func WithGuildEvents(events ...event.Type) Option {
	set := util.Set[event.Type]{}
	set.Add(events...)
	deduplicated := set.ToSlice()

	return func(st *GatewayState) error {
		if len(deduplicated) != len(events) {
			return errors.New("duplicated guild events found")
		}
		if st.intents > 0 {
			return errors.New("'GuildEvents' can not be set when using 'Intents' option")
		}

		st.guildEvents = events
		return nil
	}
}

func WithIntents(intents intent.Type) Option {
	return func(st *GatewayState) error {
		if len(st.directMessageEvents) > 0 || len(st.guildEvents) > 0 {
			return errors.New("'Intents' can not be used along with 'DirectMessageEvents' and/or 'GuildEvents'")
		}

		st.intents = intents
		return nil
	}
}

func WithShardID(id ShardID) Option {
	return func(st *GatewayState) error {
		st.shardID = id
		return nil
	}
}

func WithShardCount(count uint) Option {
	return func(st *GatewayState) error {
		st.totalNumberOfShards = count
		return nil
	}
}

func WithIdentifyConnectionProperties(properties *IdentifyConnectionProperties) Option {
	return func(st *GatewayState) error {
		st.connectionProperties = properties
		return nil
	}
}

func WithCommandRateLimiter(ratelimiter <-chan int) Option {
	return func(st *GatewayState) error {
		st.commandRateLimitChan = ratelimiter
		return nil
	}
}

func WithIdentifyRateLimiter(ratelimiter IdentifyRateLimiter) Option {
	return func(st *GatewayState) error {
		st.identifyRateLimiter = ratelimiter
		return nil
	}
}

func WithSequenceNumber(seq int64) Option {
	return func(st *GatewayState) error {
		if seq < 0 {
			return errors.New("initial sequence number can not be a negative number")
		}

		st.initialSequenceNumber = seq
		return nil
	}
}

func WithSessionID(id string) Option {
	return func(st *GatewayState) error {
		st.sessionID = id
		return nil
	}
}
