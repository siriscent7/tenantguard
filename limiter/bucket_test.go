package limiter

import (
	"testing"
	"time"
)

func TestAllowWithinCapacity(t *testing.T) {
	b := NewBucket(5, 1) // 5 tokens, refill 1/sec
	for i := 0; i < 5; i++ {
		if !b.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestDeniedWhenEmpty(t *testing.T) {
	b := NewBucket(3, 1)
	for i := 0; i < 3; i++ {
		b.Allow() // drain all tokens
	}
	if b.Allow() {
		t.Fatal("request should be denied when bucket is empty")
	}
}

func TestRefillOverTime(t *testing.T) {
	b := NewBucket(2, 10) // refills 10 tokens/sec
	b.Allow()
	b.Allow() // drained
	if b.Allow() {
		t.Fatal("should be empty immediately after draining")
	}
	time.Sleep(150 * time.Millisecond) // ~1.5 tokens refill
	if !b.Allow() {
		t.Fatal("should allow after refill")
	}
}
