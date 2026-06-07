package limiter

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisStoreAllow(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set; skipping Redis integration test")
	}

	store, err := NewRedisStore(url, 3, 0.001) // negligible refill to isolate capacity behavior
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	tenant := "test-tenant-" + t.Name()

	// clean slate
	store.client.Del(ctx, "bucket:"+tenant)

	// first 3 should pass (capacity 3)
	for i := 0; i < 3; i++ {
		ok, err := store.Allow(ctx, tenant)
		if err != nil {
			t.Fatalf("allow err: %v", err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	// 4th should be denied
	ok, _ := store.Allow(ctx, tenant)
	if ok {
		t.Fatal("4th request should be rate-limited")
	}
}

func TestRedisStoreRefill(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set; skipping Redis integration test")
	}

	// capacity 1, slow refill: 1 token per second.
	// Slow enough that network latency between requests adds < 1 token,
	// but fast enough that a deliberate wait proves refill works.
	store, err := NewRedisStore(url, 1, 1)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	tenant := "refill-tenant-" + t.Name()
	store.client.Del(ctx, "bucket:"+tenant)

	// drain the single token
	ok, _ := store.Allow(ctx, tenant)
	if !ok {
		t.Fatal("first request should be allowed")
	}
	// immediately after, should be empty (latency adds << 1 token at 1/sec)
	ok, _ = store.Allow(ctx, tenant)
	if ok {
		t.Fatal("should be rate-limited immediately after draining")
	}

	// wait well past one refill period -> at least 1 token restored
	time.Sleep(1200 * time.Millisecond)
	ok, _ = store.Allow(ctx, tenant)
	if !ok {
		t.Fatal("should be allowed after refill window")
	}
}
