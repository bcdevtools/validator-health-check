package types

import (
	"sync"
	"time"
)

type RateLimiter interface {
	Request(key string, interval time.Duration) (allowed bool)
}

var _ RateLimiter = &rateLimiter{}

type rateLimiter struct {
	sync.Mutex

	lastRequest map[string]time.Time
}

func NewRateLimiter() RateLimiter {
	return &rateLimiter{
		lastRequest: make(map[string]time.Time),
	}
}

func (r *rateLimiter) Request(key string, interval time.Duration) (allowed bool) {
	r.Lock()
	defer r.Unlock()

	if last, ok := r.lastRequest[key]; ok {
		if time.Since(last) < interval {
			return false
		}
	}

	r.lastRequest[key] = time.Now()
	return true
}
