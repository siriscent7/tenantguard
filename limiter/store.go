package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a distributed token-bucket limiter backed by Redis.
// State is shared across all server instances, so limits hold globally.
type RedisStore struct {
	client     *redis.Client
	capacity   float64
	refillRate float64
}

// NewRedisStore connects to Redis using a connection URL.
func NewRedisStore(redisURL string, capacity, refillRate float64) (*RedisStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisStore{
		client:     redis.NewClient(opt),
		capacity:   capacity,
		refillRate: refillRate,
	}, nil
}

// luaScript performs an atomic token-bucket check-and-decrement.
// All logic runs inside Redis in a single atomic operation, so concurrent
// requests from multiple server instances can't race.
//
// KEYS[1]   = bucket key (per tenant)
// ARGV[1]   = capacity
// ARGV[2]   = refill rate (tokens/sec)
// ARGV[3]   = current time (unix seconds, float)
// Returns 1 if allowed, 0 if rate-limited.
var luaScript = redis.NewScript(`
local key        = KEYS[1]
local capacity   = tonumber(ARGV[1])
local refill     = tonumber(ARGV[2])
local now        = tonumber(ARGV[3])

local data    = redis.call("HMGET", key, "tokens", "ts")
local tokens  = tonumber(data[1])
local ts      = tonumber(data[2])

if tokens == nil then
    tokens = capacity
    ts = now
end

-- lazily refill based on elapsed time
local elapsed = math.max(0, now - ts)
tokens = math.min(capacity, tokens + elapsed * refill)

local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "ts", now)
redis.call("EXPIRE", key, 3600)
return allowed
`)

// Allow checks and consumes a token for the given tenant, atomically.
func (s *RedisStore) Allow(ctx context.Context, tenantID string) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	res, err := luaScript.Run(ctx, s.client,
		[]string{"bucket:" + tenantID},
		s.capacity, s.refillRate, now,
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Close releases the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}
