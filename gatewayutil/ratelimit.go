package gatewayutil

import (
	"time"

	"github.com/beefsack/go-rate"
	"github.com/discordpkg/gateway"
)

func NewCommandRateLimiter() *LocalCommandRateLimiter {
	burstSize, duration := 120, 60*time.Second
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return &LocalCommandRateLimiter{
		rate.New(burstSize, duration),
	}
}

type LocalCommandRateLimiter struct {
	limiter *rate.RateLimiter
}

var _ gateway.RateLimiter = &LocalCommandRateLimiter{}

func (rl *LocalCommandRateLimiter) Try(_ gateway.ShardID) (bool, time.Duration) {
	return rl.limiter.Try()
}

func NewLocalIdentifyRateLimiter() *LocalIdentifyRateLimiter {
	return &LocalIdentifyRateLimiter{
		limiter: rate.New(1, 5*time.Second),
	}
}

type LocalIdentifyRateLimiter struct {
	limiter *rate.RateLimiter
}

var _ gateway.RateLimiter = &LocalIdentifyRateLimiter{}

func (rl *LocalIdentifyRateLimiter) Try(_ gateway.ShardID) (bool, time.Duration) {
	return rl.limiter.Try()
}
