package limiter

import (
	"sync"
	"time"
)

// Bucket is a thread-safe token bucket for rate limiting one tenant.
// Tokens refill continuously over time; each request consumes one token.
type Bucket struct {
	capacity   float64 // max tokens the bucket can hold
	tokens     float64 // current available tokens
	refillRate float64 // tokens added per second
	lastRefill time.Time
	mu         sync.Mutex // protects the fields above from concurrent access
}

// NewBucket creates a bucket that starts full.
func NewBucket(capacity, refillRate float64) *Bucket {
	return &Bucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow returns true if a request is permitted (a token was available),
// false if the tenant is rate-limited.
func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// refill adds tokens based on time elapsed since the last refill.
func (b *Bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now
}

// Tokens returns the current token count (useful for metrics/tests).
func (b *Bucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}
