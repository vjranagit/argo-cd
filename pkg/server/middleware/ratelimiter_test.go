package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(3, time.Second, logger)

	clientIP := "192.168.1.1"

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !rl.allow(clientIP) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should block 4th request
	if rl.allow(clientIP) {
		t.Error("Request 4 should be blocked")
	}

	// Wait for refill
	time.Sleep(time.Second + 100*time.Millisecond)

	// Should allow after refill
	if !rl.allow(clientIP) {
		t.Error("Request should be allowed after refill")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(2, time.Second, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := rl.RateLimit()(handler)

	// Test allowed requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Request %d: expected status %d, got %d", i+1, http.StatusOK, status)
		}
	}

	// Test blocked request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, status)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		forwardedFor   string
		realIP         string
		expectedIP     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:1234",
			expectedIP: "192.168.1.1",
		},
		{
			name:         "X-Real-IP",
			remoteAddr:   "192.168.1.1:1234",
			realIP:       "10.0.0.1",
			expectedIP:   "10.0.0.1",
		},
		{
			name:         "X-Forwarded-For",
			remoteAddr:   "192.168.1.1:1234",
			forwardedFor: "10.0.0.1, 192.168.1.100",
			expectedIP:   "10.0.0.1",
		},
		{
			name:         "X-Forwarded-For priority",
			remoteAddr:   "192.168.1.1:1234",
			forwardedFor: "10.0.0.1",
			realIP:       "192.168.1.100",
			expectedIP:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
