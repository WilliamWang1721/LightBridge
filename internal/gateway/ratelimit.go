package gateway

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter implements a simple token-bucket rate limiter per API key.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int           // tokens per interval
	burst   int           // max tokens
	window  time.Duration // refill interval
}

type bucket struct {
	tokens   int
	lastFill time.Time
}

func newRateLimiter(ratePerMinute, burst int) *rateLimiter {
	if ratePerMinute <= 0 {
		ratePerMinute = 60
	}
	if burst <= 0 {
		burst = ratePerMinute
	}
	return &rateLimiter{
		buckets: make(map[string]*bucket),
		rate:    ratePerMinute,
		burst:   burst,
		window:  time.Minute,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastFill: now}
		rl.buckets[key] = b
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastFill)
	if elapsed >= rl.window {
		periods := int(elapsed / rl.window)
		b.tokens += periods * rl.rate
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastFill = b.lastFill.Add(time.Duration(periods) * rl.window)
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// cleanup removes stale buckets (call periodically).
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-5 * time.Minute)
	for k, b := range rl.buckets {
		if b.lastFill.Before(cutoff) {
			delete(rl.buckets, k)
		}
	}
}

// rateLimitMiddleware wraps an http.Handler with per-key rate limiting.
// It extracts the API key from standard auth headers/query params.
func rateLimitMiddleware(rl *rateLimiter, next http.Handler) http.Handler {
	if rl == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only rate-limit API proxy paths
		if !strings.HasPrefix(r.URL.Path, "/v1/") &&
			!strings.HasPrefix(r.URL.Path, "/openai/") &&
			!strings.HasPrefix(r.URL.Path, "/openai-responses/") &&
			!strings.HasPrefix(r.URL.Path, "/gemini/") &&
			!strings.HasPrefix(r.URL.Path, "/claude/") &&
			!strings.HasPrefix(r.URL.Path, "/anthropic/") &&
			!strings.HasPrefix(r.URL.Path, "/azure/openai/") {
			next.ServeHTTP(w, r)
			return
		}

		key := extractAPIKey(r)
		if key == "" {
			key = r.RemoteAddr // fallback to IP
		}

		if !rl.allow(key) {
			w.Header().Set("content-type", "application/json")
			w.Header().Set("retry-after", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractAPIKey(r *http.Request) string {
	return clientTokenFromRequest(r)
}
