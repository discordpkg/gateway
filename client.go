package gateway

import (
	"errors"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
	"github.com/discordpkg/gateway/json"
	"io"
	"runtime"
)

var ErrOutOfSync = errors.New("sequence number was out of sync")

func NewClient(botToken string, options ...Option) (*Client, error) {
	client := &Client{
		botToken: botToken,
	}
	client.ctx = &StateCtx{client: client}

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

	// heartbeat
	if client.heartbeatHandler == nil {
		return nil, errors.New("missing heartbeat handler - use WithHeartbeatHandler")
	}

	// sharding
	if client.totalNumberOfShards == 0 {
		if client.id == 0 {
			client.totalNumberOfShards = 1
		} else {
			return nil, errors.New("missing shard count")
		}
	}
	if int(client.id) > client.totalNumberOfShards {
		return nil, errors.New("shard id is higher than shard count")
	}

	client.ctx.state = &HelloState{
		StateCtx: client.ctx,
		Identity: &Identify{ // TODO: re-use for resumes
			BotToken:       botToken,
			Properties:     &client.connectionProperties,
			Compress:       false,
			LargeThreshold: 0,
			Shard:          [2]int{int(client.id), client.totalNumberOfShards},
			Presence:       nil,
			Intents:        client.intents,
		},
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

	ctx                  *StateCtx
	commandRateLimiter   CommandRateLimiter
	identifyRateLimiter  IdentifyRateLimiter
	heartbeatHandler     HeartbeatHandler
	connectionProperties interface{}
	totalNumberOfShards  int
}

func (c *Client) ResumeDetails() (resumeGatewayURL string, sessionID string, err error) {
	if st, ok := c.ctx.state.(*ResumableClosedState); ok {
		return st.ResumeGatewayURL, st.SessionID, nil
	}
	return "", "", errors.New("not a resumable state")
}

func (c *Client) Close(pipe io.Writer) error {
	return c.ctx.WriteNormalClose(pipe)
}

func (c *Client) ProcessNextPayload(payload *Payload, pipe io.Writer) (err error) {
	if c.ctx.sequenceNumber.CompareAndSwap(payload.Seq-1, payload.Seq) {
		return c.ctx.Process(payload, pipe)
	} else if c.ctx.sequenceNumber.Load() >= payload.Seq {
		// already handled
		return nil
	}

	c.ctx.state = &ClosedState{}
	return ErrOutOfSync
}

func (c *Client) Write(pipe io.Writer, opc command.Type, payload json.RawMessage) error {
	return c.ctx.Write(pipe, opc, payload)
}
