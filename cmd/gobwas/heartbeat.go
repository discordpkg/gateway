package discordgatewaygobwas

import (
	"context"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway"
)

type heart struct {
	interval          time.Duration
	conn              net.Conn
	shard             *discordgateway.ClientState
	forcedReadTimeout *atomic.Bool
	gotAck            atomic.Bool
}

func (h *heart) pulser(ctx context.Context) {
	// shard <-> pulser

	writer := wsutil.NewWriter(h.conn, ws.StateClientSide, ws.OpText)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	logrus.Debugf("created heartbeat ticker with interval %s", h.interval)
loop:
	select {
	case <-ctx.Done():
		logrus.Debug("heartbeat pulser timed out")
	case <-ticker.C:
		if h.gotAck.CAS(true, false) {
			if err := h.shard.Heartbeat(writer); err != nil {
				logrus.Error("failed to send heartbeat. ", err)
			} else {
				logrus.Debug("sent heartbeat")
				goto loop // go back to start
			}
		} else {
			logrus.Info("have not received heartbeat, shutting down")
		}
	}
	if h.shard.Closed() {
		// it was closed by the main go routine for this shard
		// so it should not be handing on read anymore
		return
	}

	plannedTimeoutWindow := 5 * time.Second
	if err := writeClose(h.conn, h.shard, "heart beat failure"); err != nil {
		plannedTimeoutWindow = 100 * time.Millisecond
	}

	// handle network connection loss
	logrus.Debug("started fallback for connection issues")
	<-time.After(plannedTimeoutWindow)
	h.forcedReadTimeout.Store(true)
	logrus.Info("setting read deadline")
	if err := h.conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		logrus.Error("failed to set read deadline", err)
	}
}
