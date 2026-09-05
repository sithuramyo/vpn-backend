package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"vpn-backend/pkg/response"
)

// RateLimiter applies a per-client-IP token bucket. A single VPS running
// everything on one process makes an in-memory limiter sufficient — no
// Redis needed for this scale.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (rl *RateLimiter) limiterFor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, ok := rl.limiters[key]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[key] = l
	}
	return l
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := rl.limiterFor(c.ClientIP())
		if !limiter.Allow() {
			response.Error(c, 429, "TOO_MANY_REQUESTS", "Too many requests. Please slow down.")
			return
		}
		c.Next()
	}
}

// StartCleanup periodically drops idle per-IP limiters so the map doesn't
// grow without bound.
func (rl *RateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			rl.limiters = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		}
	}()
}
