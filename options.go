package gateway

import (
	"errors"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
)

// Option for initializing a new gateway client. An option must be deterministic regardless
// of when or how many times it is executed.
type Option func(client *Client) error

var noopOption = func(_ *Client) error {
	return nil
}

func WithBotToken(token string) Option {
	return func(client *Client) error {
		client.botToken = token
		return nil
	}
}

func WithDirectMessageEvents(events ...event.Type) Option {
	set := util.Set[event.Type]{}
	set.Add(events...)
	deduplicated := set.ToSlice()

	return func(client *Client) error {
		if len(deduplicated) != len(events) {
			return errors.New("duplicated direct message events found")
		}

		client.intents |= intent.DMEventsToIntents(deduplicated)
		client.allowlist.Add(deduplicated...) // apply prune optimization
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

		client.intents |= intent.GuildEventsToIntents(deduplicated)
		client.allowlist.Add(deduplicated...) // apply prune optimization
		return nil
	}
}

func WithIntents(intents intent.Type) Option {
	return func(client *Client) error {
		if client.allowlist != nil {
			return errors.New("'Intents' can not be used along with 'DirectMessageEvents' and/or 'GuildEvents'")
		}

		client.intents = intents
		client.allowlist.Add(intent.Events(intents)...)
		return nil
	}
}

func WithShardInfo(id ShardID, count int) Option {
	if count < 0 {
		panic("shard count must be above 0")
	}

	return func(client *Client) error {
		client.id = id
		client.totalNumberOfShards = count
		return nil
	}
}

func WithExistingSession(deadClient *Client) Option {
	if deadClient == nil {
		return noopOption
	}

	return func(client *Client) error {
		st, ok := deadClient.ctx.state.(*ResumableClosedState)
		if !ok {
			// panic("the existing client did not have a valid session saved")
			// TODO: is this bad form?
			return nil
		}

		client.ctx.SessionID = st.ctx.SessionID
		client.ctx.ResumeGatewayURL = st.ctx.ResumeGatewayURL
		client.ctx.sequenceNumber.Store(st.ctx.sequenceNumber.Load())

		client.ctx.SetState(&ResumeState{&ConnectedState{ctx: client.ctx}})
		return nil
	}
}

func WithIdentifyConnectionProperties(properties *IdentifyConnectionProperties) Option {
	return func(client *Client) error {
		client.connectionProperties = properties
		return nil
	}
}

func WithCommandRateLimiter(ratelimiter RateLimiter) Option {
	return func(client *Client) error {
		client.commandRateLimiter = ratelimiter
		return nil
	}
}

func WithIdentifyRateLimiter(ratelimiter RateLimiter) Option {
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

// WithEventHandler provides a callback that is triggered on incoming events. Note that the allowlist will filter
// out events you have not requested.
//
// Warning: this function call is blocking. You should not run heavy logic in the handler, preferably just forward it
// to a processing component. An example usage would be to send it to a buffered worker channel.
func WithEventHandler(handler Handler) Option {
	return func(client *Client) error {
		client.eventHandler = handler
		return nil
	}
}

func WithLogger(logger Logger) Option {
	return func(client *Client) error {
		client.logger = logger
		return nil
	}
}
