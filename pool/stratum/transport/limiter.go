package transport

import (
	"math"
	"time"
)

type tokenBucket struct {
	tokens   float64
	rate     float64
	capacity float64
	last     time.Time
}

func newTokenBucket(rate, burst uint32, now time.Time) *tokenBucket {
	return &tokenBucket{tokens: float64(burst), rate: float64(rate), capacity: float64(burst), last: now}
}

func (b *tokenBucket) Allow(now time.Time) bool {
	if now.After(b.last) {
		b.tokens = math.Min(b.capacity, b.tokens+now.Sub(b.last).Seconds()*b.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
