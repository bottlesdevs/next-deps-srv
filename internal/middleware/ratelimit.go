package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

type RateLimiter struct {
	mu      sync.RWMutex
	buckets sync.Map
	cfg     models.RateLimitConfig
}

func NewRateLimiter(cfg models.RateLimitConfig) *RateLimiter {
	return &RateLimiter{cfg: cfg}
}

func (rl *RateLimiter) UpdateConfig(cfg models.RateLimitConfig) {
	rl.mu.Lock()
	rl.cfg = cfg
	rl.mu.Unlock()
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.RLock()
	cfg := rl.cfg
	rl.mu.RUnlock()

	if !cfg.Enabled {
		return true
	}

	rate := float64(cfg.RequestsPerMinute) / 60.0
	burst := float64(cfg.BurstSize)
	if burst < 1 {
		burst = 1
	}

	v, _ := rl.buckets.LoadOrStore(ip, &bucket{tokens: burst, lastSeen: time.Now()})
	b := v.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.lastSeen = now
	b.tokens += elapsed * rate
	if b.tokens > burst {
		b.tokens = burst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func RateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ip = xff
			}
			if !rl.allow(ip) {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
