package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRateLimit_Allowed(t *testing.T) {
	rl := middleware.NewRateLimiter(models.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         5,
	})
	h := middleware.RateLimit(rl)(http.HandlerFunc(okHandler))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, w.Code)
		}
	}
}

func TestRateLimit_Exceeded(t *testing.T) {
	rl := middleware.NewRateLimiter(models.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         2,
	})
	h := middleware.RateLimit(rl)(http.HandlerFunc(okHandler))

	codes := make([]int, 5)
	for i := range codes {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		codes[i] = w.Code
	}
	got429 := false
	for _, c := range codes {
		if c == http.StatusTooManyRequests {
			got429 = true
		}
	}
	if !got429 {
		t.Error("expected at least one 429 after burst exhaustion")
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	rl := middleware.NewRateLimiter(models.RateLimitConfig{Enabled: false})
	h := middleware.RateLimit(rl)(http.HandlerFunc(okHandler))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:0"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("disabled rate limiter blocked request %d", i)
		}
	}
}

func TestRateLimit_UpdateConfig(t *testing.T) {
	rl := middleware.NewRateLimiter(models.RateLimitConfig{Enabled: false})
	rl.UpdateConfig(models.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
	})
	h := middleware.RateLimit(rl)(http.HandlerFunc(okHandler))

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "5.6.7.8:0"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request should pass: %d", w1.Code)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "5.6.7.8:0"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be rate limited: %d", w2.Code)
	}
}
