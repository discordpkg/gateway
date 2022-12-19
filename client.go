package gateway

import (
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/encoding"
	"io"
	"runtime"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/internal/util"
)

var ErrOutOfSync = errors.New("sequence number was out of sync")
var ErrNotConnectedYet = errors.New("client is not in a connected state")

func NewClient(options ...Option) (*Client, error) {
	client := &Client{
		allowlist: util.Set[event.Type]{},
		logger:    &nopLogger{},
	}
	client.ctx = &StateCtx{client: client}

	for i := range options {
		if err := options[i](client); err != nil {
			return nil, err
		}
	}
	client.ctx.logger = client.logger // ugh..

	if client.botToken == "" {
		return nil, errors.New("missing bot token")
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
		client.ctx.SetState(&HelloState{
			ctx: client.ctx,
			Identity: &Identify{ // TODO: re-use for resumes
				BotToken:       client.botToken,
				Properties:     &client.connectionProperties,
				Compress:       false,
				LargeThreshold: 0,
				Shard:          [2]int{int(client.id), client.totalNumberOfShards},
				Presence:       nil,
				Intents:        client.intents,
			},
		})
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

	commandRateLimiter  RateLimiter
	identifyRateLimiter RateLimiter

	heartbeatHandler HeartbeatHandler

	ctx    *StateCtx
	logger Logger
}

func (c *Client) String() string {
	data := ""
	data += fmt.Sprintln(fmt.Sprintf("shard %d out of %d shards", c.id, c.totalNumberOfShards))
	data += fmt.Sprintln("intents:", c.intents)
	data += fmt.Sprintln("events:", c.intents)
	return data
}

// ResumeURL returns the URL to be used when dialing a new websocket connection. An empty string
// is returned when the shard can not be resumed, and you should instead use "Get Gateway Bot" endpoint to fetch
// the correct URL for connecting.
//
// The client is assumed to have been correctly closed before calling this.
func (c *Client) ResumeURL() string {
	if _, ok := c.ctx.state.(*ResumableClosedState); ok {
		return c.ctx.ResumeGatewayURL
	}

	return ""
}

func (c *Client) Close(closeWriter io.Writer) error {
	return c.ctx.Close(closeWriter)
}

func (c *Client) read(client io.Reader) (*Payload, int, error) {
	data, err := io.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &Payload{}
	if err = encoding.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	return packet, len(data), nil
}

func (c *Client) process(payload *Payload, pipe io.Writer) (err error) {
	// we consider 0 to be either the first message or messages without sequence numbers such as heartbeat ack
	if c.ctx.sequenceNumber.Load() == 0 || c.ctx.sequenceNumber.CompareAndSwap(payload.Seq-1, payload.Seq) {
		return c.ctx.Process(payload, pipe)
	} else if c.ctx.sequenceNumber.Load() >= payload.Seq {
		// already handled
		return nil
	}

	c.ctx.SetState(&ClosedState{})
	return ErrOutOfSync
}

func (c *Client) ProcessNext(reader io.Reader, writer io.Writer) (*Payload, error) {
	payload, _, err := c.read(reader)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("processing payload: %s", payload)
	return payload, c.process(payload, writer)
}

func (c *Client) Write(pipe io.Writer, evt event.Type, payload encoding.RawMessage) error {
	if _, ok := c.ctx.state.(*ConnectedState); !ok {
		return ErrNotConnectedYet
	}

	return c.ctx.Write(pipe, evt, payload)
}
