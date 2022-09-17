package gateway

import (
	"github.com/beefsack/go-rate"
	"time"
)

func NewCommandRateLimiter() CommandRateLimiter {
	burstSize, duration := 120, 60*time.Second
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return rate.New(burstSize, duration)
}

func NewLocalIdentifyRateLimiter() IdentifyRateLimiter {
	return &LocalIdentifyRateLimiter{
		limiter: rate.New(1, 5*time.Second),
	}
}

type LocalIdentifyRateLimiter struct {
	limiter *rate.RateLimiter
}

func (rl *LocalIdentifyRateLimiter) Try(_ ShardID) (bool, time.Duration) {
	return rl.limiter.Try()
}
