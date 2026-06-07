package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/siriscent7/tenantguard/gateway"
	"github.com/siriscent7/tenantguard/limiter"
)

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	// 10 tokens capacity, refill 5/sec per tenant (tune as you like)
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
	mux.Handle("/api", gateway.RateLimit(store, api))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "healthy")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("TenantGuard listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))

	_ = context.Background() // (kept import tidy; remove if unused warning)
}
