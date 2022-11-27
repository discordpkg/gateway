package gateway

import (
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/discordpkg/gateway/command"
)

type HeartbeatHandler interface {
	Configure(ctx *StateCtx, interval time.Duration)
	Run()
}

type DefaultHeartbeatHandler struct {
	TextWriter  io.Writer
	CloseWriter io.WriteCloser

	ctx      *StateCtx
	interval time.Duration
}

func (p *DefaultHeartbeatHandler) Configure(ctx *StateCtx, interval time.Duration) {
	if p.TextWriter == nil {
		panic("heartbeat handler: missing text writer")
	}
	if p.CloseWriter == nil {
		panic("heartbeat handler: missing close writer")
	}

	p.ctx = ctx
	p.interval = interval
}

func (p *DefaultHeartbeatHandler) Run() {
	jitter := float64(rand.Intn(100)) / 100
	initialDelay := p.interval.Seconds() * jitter
	<-time.After(time.Duration(initialDelay) * time.Second)

	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		if !p.ctx.heartbeatACK.CompareAndSwap(true, false) {
			// did not receive heart beat ack since last heartbeat, shutting down
			break
		}

		seq := p.ctx.sequenceNumber.Load()
		seqStr := strconv.FormatInt(seq, 10)
		if err := p.ctx.Write(p.TextWriter, command.Heartbeat, []byte(seqStr)); err != nil {
			// failed to send heartbeat, shutting down
			break
		}

		select {
		case <-timer.C:
			if p.ctx.closed.Load() {
				return
			}
		}
	}

	// TODO: shutdown websocket somehow
}
