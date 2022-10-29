package gateway

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
	"github.com/discordpkg/gateway/json"
	"github.com/discordpkg/gateway/opcode"
)

var ErrOutOfSync = errors.New("sequence number was out of sync")

type Write func(opc command.Type, payload json.RawMessage) error

func NewClient(botToken string, options ...Option) (*Client, error) {
	client := &Client{
		botToken: botToken,
	}

	for i := range options {
		if err := options[i](client); err != nil {
			return nil, err
		}
	}

	if client.intents == 0 && (len(client.guildEvents) > 0 || len(client.directMessageEvents) > 0) {
		// derive intents
		client.intents |= intent.GuildEventsToIntents(client.guildEvents)
		client.intents |= intent.DMEventsToIntents(client.directMessageEvents)

		// whitelisted events specified events only
		client.whitelist = util.Set[event.Type]{}
		client.whitelist.Add(client.guildEvents...)
		client.whitelist.Add(client.directMessageEvents...)

		// crucial for normal function
		client.whitelist.Add(event.Ready, event.Resumed)
	}

	// rate limits
	if client.commandRateLimiter == nil {
		return nil, errors.New("missing command rate limiter - try 'gatewayutil.NewCommandRateLimiter()'")
	}
	if client.identifyRateLimiter == nil {
		return nil, errors.New("missing identify rate limiter - try 'gatewayutil.NewLocalIdentifyRateLimiter()'")
	}

	// connection properties
	if client.connectionProperties == nil {
		client.connectionProperties = &IdentifyConnectionProperties{
			OS:      runtime.GOOS,
			Browser: "github.com/discordpkg/gateway",
			Device:  "github.com/discordpkg/gateway",
		}
	}

	// sharding
	if client.totalNumberOfShards == 0 {
		if client.id == 0 {
			client.totalNumberOfShards = 1
		} else {
			return nil, errors.New("missing shard count")
		}
	}
	if uint(client.id) > client.totalNumberOfShards {
		return nil, errors.New("shard id is higher than shard count")
	}

	return client, nil
}

type Client struct {
	botToken string
	id       ShardID

	// TODO: cleanup

	// events that are not found in the whitelist are viewed as redundant and are
	// skipped / ignored
	whitelist           util.Set[event.Type]
	directMessageEvents []event.Type
	guildEvents         []event.Type

	intents intent.Type

	ctx                 *StateCtx
	state               State
	commandRateLimiter  CommandRateLimiter
	identifyRateLimiter IdentifyRateLimiter
}

func (c *Client) ProcessNextPayload(payload *Payload, pipe io.Writer) (err error) {
	if c.ctx.sequenceNumber.CAS(payload.Seq-1, payload.Seq) {
		c.state, err = c.state.Process(payload, c.write(pipe))
		switch c.state.(type) {
		case *ClosedState:
			c.ctx.closed.Store(true)
		}

		return err
	} else if c.ctx.sequenceNumber.Load() >= payload.Seq {
		// already handled
		return nil
	}

	c.state = &ClosedState{}
	return ErrOutOfSync
}

func (c *Client) write(pipe io.Writer) func(opc command.Type, payload json.RawMessage) error {
	return func(opc command.Type, payload json.RawMessage) (err error) {
		// heartbeat should always be sent.
		// Try reserving some calls for heartbeats when you configure your rate limiter.
		if opc != command.Heartbeat {
			if ok, timeout := c.commandRateLimiter.Try(); !ok {
				<-time.After(timeout)
			}
		}
		if opc == command.Identify {
			if available, _ := c.identifyRateLimiter.Try(c.id); !available {
				return errors.New("identify rate limiter denied shard to identify")
			}
		}

		packet := Payload{
			Op:   opcode.Type(opc),
			Data: payload,
		}

		var data []byte
		data, err = json.Marshal(&packet)
		if err != nil {
			return fmt.Errorf("unable to marshal packet; %w", err)
		}

		_, err = pipe.Write(data)
		return err
	}
}
