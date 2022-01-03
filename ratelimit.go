package discordgateway

import (
	"github.com/bradfitz/iter"
	"io"
	"sync"
	"time"
)

type channelTaker struct {
	c <-chan int
}

func (c *channelTaker) Take(_ ShardID) bool {
	if c.c != nil {
		select {
		case _, ok := <-c.c:
			if ok {
				return true
			}
		}
	}
	return false
}

type channelCloser struct {
	mu     sync.Mutex
	c      chan int
	closed bool
}

func (c *channelCloser) Closed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *channelCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c != nil {
		close(c.c)
		c.closed = true
	}
	return nil
}

func NewCommandRateLimiter() (<-chan int, io.Closer) {
	burstSize, duration := 120, 60*time.Second
	burstSize -= 4 // reserve 4 calls for heartbeat
	burstSize -= 1 // reserve one call, in case discord requests a heartbeat

	return NewRateLimiter(burstSize, duration)
}

func NewIdentifyRateLimiter() (<-chan int, io.Closer) {
	return NewRateLimiter(1, 5*time.Second)
}

func NewRateLimiter(burstCapacity int, burstDuration time.Duration) (<-chan int, io.Closer) {
	c := make(chan int, burstCapacity)
	closer := &channelCloser{c: c}
	refill := func() {
		burstSize := burstCapacity - len(c)

		closer.mu.Lock()
		defer closer.mu.Unlock()
		if closer.closed {
			return
		}

		for range iter.N(burstSize) {
			c <- 0
		}
	}

	go func() {
		t := time.NewTicker(burstDuration)
		defer t.Stop()

		for {
			<-t.C
			if closer.Closed() {
				// channel has been closed
				break
			}

			refill()
		}
	}()

	refill()
	return c, closer
}
