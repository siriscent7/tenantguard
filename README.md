# 🛡️ TenantGuard

**A multi-tenant rate-limiting control plane that keeps one noisy tenant from starving the rest.**

Distributed per-tenant token-bucket rate limiting with atomic Redis-backed
counters, so limits hold correctly across multiple server instances.

🔗 **Live demo:** https://tenantguard-bvid.onrender.com

---

## The problem
In multi-tenant SaaS, many customers share one backend. One tenant flooding the
API can starve everyone else — the **noisy-neighbor problem**. TenantGuard
enforces fair, per-tenant rate limits so no single tenant can monopolize shared
resources.

## What it does
- **Per-tenant token-bucket limiting** — each tenant gets an independent quota.
- **Distributed & consistent** — bucket state lives in Redis; an atomic Lua
  script makes check-and-decrement race-free across multiple server instances.
- **HTTP gateway middleware** — drop-in middleware returns `429 Too Many
  Requests` (with a `Retry-After` header) when a tenant exceeds its limit.
- **Lazy refill** — tokens accrue on-demand based on elapsed time (no background
  timers), so it scales cheaply to many tenants.

## Tenant isolation in action
=== Noisy tenant 'flooder' — 30 concurrent requests ===
flooder: 10 served, 20 throttled
=== Well-behaved tenant 'goodguy' — 5 concurrent requests ===
goodguy: 5 served, 0 throttled

The flooder hits its limit; the well-behaved tenant is completely unaffected.

## Benchmark
Load-tested with **k6** — 50 concurrent virtual users across 5 tenants for 15s:
- **2,340 requests handled (~153 req/s), zero errors** — every response was a
  valid `200` or `429`.
- **~82% of requests correctly throttled** (`429`): with 50 users sharing 5
  tenant quotas, the limiter rejected excess traffic exactly as designed while
  serving every in-quota request.
- **Median latency ~285 ms**, bounded by the round-trip to managed Redis
  (Upstash). The gateway's own decision overhead is negligible.

The system stays **stable and correct under heavy concurrent load** — it protects
tenant quotas rather than failing open.

## Architecture
```
HTTP request (X-Tenant-ID header)
│
▼
RateLimit middleware ── extract tenant → check limiter
│
▼
RedisStore.Allow() ── atomic Lua: refill + check + decrement (in Redis)
│
├── allowed → forward to API handler (200)
└── denied → 429 Too Many Requests
```

## Run locally
```bash
git clone https://github.com/siriscent7/tenantguard.git
cd tenantguard
REDIS_URL="rediss://<your-upstash-url>" go run main.go

# in another terminal:
curl -i -H "X-Tenant-ID: acme" localhost:8080/api
```

## Demo Scripts
```bash
# concurrent flood test for one tenant
./loadtest/flood.sh acme 20

# noisy-neighbor isolation demo
./loadtest/noisy_neighbor.sh
```

## Tests
```bash
# in-memory unit tests
go test ./limiter -v

# + Redis integration tests (requires a Redis URL)
REDIS_URL="rediss://..." go test ./limiter -v
```

5 tests: token-bucket capacity & refill (in-memory) + distributed
capacity & refill (Redis integration).

## Tech stack
Go · Redis (Upstash) · atomic Lua scripting · k6 · Docker · Render

## Limitations
- Fixed per-tenant limits (not yet configurable per-tenant at runtime).
- Distributed mode depends on Redis; single-instance mode can use the in-memory limiter.
- Latency in the live demo is dominated by the network round-trip to managed Redis; co-locating Redis would reduce it significantly.

## Future Work
- Per-tenant configurable quotas via a config endpoint.
- Prometheus metrics + Grafana dashboard for live usage.
- Tiered plans (different limits per subscription tier).
- A /usage endpoint exposing per-tenant consumption.

