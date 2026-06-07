package gateway

import (
	"context"
	"net/http"
)

// Limiter is the behavior the gateway needs: decide if a tenant may proceed.
// Both our in-memory and Redis limiters can satisfy this (we'll wire Redis).
type Limiter interface {
	Allow(ctx context.Context, tenantID string) (bool, error)
}

// RateLimit wraps an http.Handler, enforcing per-tenant limits before
// letting requests through.
func RateLimit(limiter Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
			return
		}

		allowed, err := limiter.Allow(r.Context(), tenantID)
		if err != nil {
			http.Error(w, "rate limiter error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded for tenant "+tenantID,
				http.StatusTooManyRequests) // 429
			return
		}

		next.ServeHTTP(w, r) // allowed -> pass to the real handler
	})
}
