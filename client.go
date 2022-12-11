package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
	"github.com/discordpkg/gateway/json"
	"io"
	"runtime"
)

var ErrOutOfSync = errors.New("sequence number was out of sync")
var ErrNotConnectedYet = errors.New("client is not in a connected state")

func NewClient(botToken string, options ...Option) (*Client, error) {
	client := &Client{
		botToken:  botToken,
		allowlist: util.Set[event.Type]{},
	}
	client.ctx = &StateCtx{client: client}

	for i := range options {
		if err := options[i](client); err != nil {
			return nil, err
		}
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

	if client.ctx.state == nil {
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
	}
	return client, nil
}

// Client provides a user target interface, for simplified Discord interaction.
//
// Note: It's not suitable for internal processes/states.
type Client struct {
	botToken             string
	id                   ShardID
	totalNumberOfShards  int
	connectionProperties interface{}
	intents              intent.Type

	allowlist    util.Set[event.Type]
	eventHandler Handler

	commandRateLimiter  CommandRateLimiter
	identifyRateLimiter IdentifyRateLimiter

	heartbeatHandler HeartbeatHandler

	ctx *StateCtx
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

func (c *Client) read(client io.Reader) (*Payload, int, error) {
	data, err := io.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &Payload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	return packet, len(data), nil
}

func (c *Client) process(payload *Payload, pipe io.Writer) (err error) {
	if c.ctx.sequenceNumber.CompareAndSwap(payload.Seq-1, payload.Seq) {
		return c.ctx.Process(payload, pipe)
	} else if c.ctx.sequenceNumber.Load() >= payload.Seq {
		// already handled
		return nil
	}

	c.ctx.state = &ClosedState{}
	return ErrOutOfSync
}

func (c *Client) ProcessNext(reader io.Reader, writer io.Writer) (*Payload, error) {
	payload, _, err := c.read(reader)
	if err != nil {
		return nil, err
	}

	return payload, c.process(payload, writer)
}

func (c *Client) Write(pipe io.Writer, opc command.Type, payload json.RawMessage) error {
	if _, ok := c.ctx.state.(*ConnectedState); !ok {
		return ErrNotConnectedYet
	}

	return c.ctx.Write(pipe, opc, payload)
}
