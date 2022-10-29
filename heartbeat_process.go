package gateway

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/discordpkg/gateway/command"
)

type HeartbeatProcess struct {
	*StateCtx
	interval time.Duration
}

func (p *HeartbeatProcess) Run(write Write) {
	jitter := float64(rand.Intn(100)) / 100
	<-time.After(time.Duration(p.interval.Seconds()*jitter) * time.Second)

	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		if !p.heartbeatACK.CAS(true, false) {
			// did not receive heart beat ack since last heartbeat, shutting down
			break
		}

		seq := p.sequenceNumber.Load()
		seqStr := strconv.FormatInt(seq, 10)
		if err := write(command.Heartbeat, []byte(seqStr)); err != nil {
			// failed to send heartbeat, shutting down
			break
		}

		select {
		case <-timer.C:
			if p.closed.Load() {
				return
			}
		}
	}

	// TODO: shutdown websocket somehow
}
