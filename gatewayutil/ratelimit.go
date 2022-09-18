package gatewayutil

import (
	"github.com/beefsack/go-rate"
	"github.com/discordpkg/gateway"
	"time"
)

func NewCommandRateLimiter() gateway.CommandRateLimiter {
	burstSize, duration := 120, 60*time.Second
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return rate.New(burstSize, duration)
}

func NewLocalIdentifyRateLimiter() gateway.IdentifyRateLimiter {
	return &LocalIdentifyRateLimiter{
		limiter: rate.New(1, 5*time.Second),
	}
}

type LocalIdentifyRateLimiter struct {
	limiter *rate.RateLimiter
}

func (rl *LocalIdentifyRateLimiter) Try(_ gateway.ShardID) (bool, time.Duration) {
	return rl.limiter.Try()
}
