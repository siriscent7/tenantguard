package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/siriscent7/tenantguard/gateway"
	"github.com/siriscent7/tenantguard/limiter"
)

const landingHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>TenantGuard</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; background:#0f172a;
         color:#e2e8f0; margin:0; padding:40px; line-height:1.6; }
  .wrap { max-width:680px; margin:0 auto; }
  h1 { font-size:2rem; margin-bottom:4px; }
  .tag { color:#94a3b8; margin-bottom:32px; }
  .card { background:#1e293b; border:1px solid #334155; border-radius:10px;
          padding:20px 24px; margin-bottom:20px; }
  code { background:#0f172a; padding:2px 8px; border-radius:5px;
         font-family: ui-monospace, monospace; color:#7dd3fc; }
  pre { background:#0f172a; padding:14px; border-radius:8px; overflow-x:auto;
        font-size:0.9rem; }
  .pill { display:inline-block; background:#16a34a; color:#fff; font-size:0.75rem;
          padding:2px 10px; border-radius:999px; margin-left:8px; vertical-align:middle; }
  a { color:#7dd3fc; }
</style>
</head>
<body>
  <div class="wrap">
    <h1>🛡️ TenantGuard <span class="pill">LIVE</span></h1>
    <p class="tag">A multi-tenant rate-limiting control plane that keeps one
       noisy tenant from starving the rest.</p>

    <div class="card">
      <strong>What it does</strong>
      <p>Distributed per-tenant token-bucket rate limiting, backed by Redis with
         an atomic Lua script so limits hold correctly across multiple server
         instances. Exceed your quota and you get <code>429 Too Many Requests</code>.</p>
    </div>

    <div class="card">
      <strong>Try it</strong>
      <pre># health check
curl ` + `THIS_URL` + `/health

# a request as tenant "acme"
curl -i -H "X-Tenant-ID: acme" ` + `THIS_URL` + `/api

# flood it — watch 200s turn into 429s
for i in $(seq 1 20); do
  curl -s -o /dev/null -w "%{http_code} " \
    -H "X-Tenant-ID: flooder" ` + `THIS_URL` + `/api &
done; wait; echo</pre>
    </div>

    <div class="card">
      <strong>Stack:</strong> Go · Redis (Upstash) · atomic Lua · Docker · Render<br>
      <strong>Source:</strong>
      <a href="https://github.com/siriscent7/tenantguard">github.com/siriscent7/tenantguard</a>
    </div>
  </div>
</body>
</html>`

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	// capacity 5 tokens, refill 1/sec per tenant
	store, err := limiter.NewRedisStore(redisURL, 5, 1)
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	defer store.Close()

	// the "real" API handler that rate limiting protects
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := r.Header.Get("X-Tenant-ID")
		fmt.Fprintf(w, "OK - request served for tenant %s\n", tenant)
	})

	mux := http.NewServeMux()

	// rate-limited API endpoint
	mux.Handle("/api", gateway.RateLimit(store, api))

	// health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "healthy")
	})

	// styled landing page at root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r) // keep 404 for unknown paths
			return
		}
		scheme := "https"
		if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
			scheme = "http"
		}
		page := strings.ReplaceAll(landingHTML, "THIS_URL", scheme+"://"+r.Host)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("TenantGuard listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
