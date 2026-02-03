package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // requests per interval
	interval time.Duration // time window
	logger   *slog.Logger
}

type bucket struct {
	tokens       int
	lastRefill   time.Time
	mu           sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: maximum requests allowed per interval
// interval: time window (e.g., 1 minute)
func NewRateLimiter(rate int, interval time.Duration, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		logger:   logger.With("component", "ratelimiter"),
	}

	// Start cleanup goroutine to remove stale buckets
	go rl.cleanup()

	return rl
}

// RateLimit returns a middleware that enforces rate limiting per client IP
func (rl *RateLimiter) RateLimit() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			if !rl.allow(clientIP) {
				rl.logger.Warn("rate limit exceeded",
					"client_ip", clientIP,
					"path", r.URL.Path,
				)
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.interval.Seconds())))
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request from the given client should be allowed
func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	b, exists := rl.buckets[clientIP]
	if !exists {
		b = &bucket{
			tokens:     rl.rate,
			lastRefill: time.Now(),
		}
		rl.buckets[clientIP] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	
	if elapsed >= rl.interval {
		// Full refill
		b.tokens = rl.rate
		b.lastRefill = now
	} else {
		// Partial refill based on elapsed time
		tokensToAdd := int(float64(rl.rate) * (elapsed.Seconds() / rl.interval.Seconds()))
		b.tokens = min(b.tokens+tokensToAdd, rl.rate)
		if tokensToAdd > 0 {
			b.lastRefill = now
		}
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes stale buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.interval * 2)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > rl.interval*2 {
				delete(rl.buckets, ip)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
