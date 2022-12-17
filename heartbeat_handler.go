package gateway

import (
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/discordpkg/gateway/event"
)

type HeartbeatHandler interface {
	Configure(ctx *StateCtx, interval time.Duration)
	Run()
}

type DefaultHeartbeatHandler struct {
	TextWriter io.Writer

	// ConnectionCloser assumes that by closing the connection we can trigger an interrupt signal for whatever process
	// that is busy reading/waiting for the next websocket frame/message. After an interrupt you are expected to call
	// Client.Close - allowing the client to properly update its internal state. This allows the client to be operated
	// be a single process, avoiding the need of locking complexity (see gatewayutil/shard.go for an example).
	//
	// If this doesn't achieve what you need/want, then implement you own version using the HeartbeatHandler interface.
	ConnectionCloser io.Closer

	ctx      *StateCtx
	interval time.Duration
}

func (p *DefaultHeartbeatHandler) Configure(ctx *StateCtx, interval time.Duration) {
	if p.TextWriter == nil {
		panic("heartbeat handler: missing text writer")
	}
	if p.ConnectionCloser == nil {
		panic("heartbeat handler: missing close writer")
	}

	p.ctx = ctx
	p.interval = interval
}

func (p *DefaultHeartbeatHandler) Run() {
	jitter := float64(rand.Intn(100)) / 100
	initialDelay := time.Duration(p.interval.Seconds()*jitter) * time.Second

	p.ctx.logger.Debug("heartbeat process waiting %s before first heartbeat write", initialDelay)
	<-time.After(initialDelay)

	p.ctx.logger.Debug("configured heartbeat process to run every %s", p.interval)
	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		if !p.ctx.heartbeatACK.CompareAndSwap(true, false) {
			p.ctx.logger.Info("did not receive heart beat ack since last heartbeat")
			break
		}

		seq := p.ctx.sequenceNumber.Load()
		seqStr := strconv.FormatInt(seq, 10)
		if err := p.ctx.Write(p.TextWriter, event.Heartbeat, []byte(seqStr)); err != nil {
			p.ctx.logger.Info("unable to send heartbeat: %s", err.Error())
			break
		}

		select {
		case <-timer.C:
			if p.ctx.closed.Load() {
				p.ctx.logger.Info("state context was marked closed, stopping heartbeat process")
				return
			}
		}
	}

	p.ctx.logger.Debug("closing heartbeat process")
	_ = p.ConnectionCloser.Close()
}
