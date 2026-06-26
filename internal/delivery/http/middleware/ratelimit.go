package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory fixed-window limiter keyed by client IP — the
// Better Auth default storage strategy. Swap for a Redis/DB-backed store for
// horizontal scaling.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	max     int
	window  time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter builds a limiter allowing max requests per window per key.
func NewRateLimiter(max int, win time.Duration) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]*window),
		max:     max,
		window:  win,
	}
}

// Middleware enforces the limit, returning 429 with Retry-After when exceeded.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, remaining, retryAfter := rl.allow(c.ClientIP())
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.max))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) allow(key string) (allowed bool, remaining int, retryAfter time.Duration) {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, ok := rl.windows[key]
	if !ok || now.After(w.resetAt) {
		rl.windows[key] = &window{count: 1, resetAt: now.Add(rl.window)}
		return true, rl.max - 1, 0
	}
	if w.count >= rl.max {
		return false, 0, time.Until(w.resetAt)
	}
	w.count++
	return true, rl.max - w.count, 0
}

// Cleanup periodically evicts expired windows until ctx is cancelled.
func (rl *RateLimiter) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			rl.mu.Lock()
			for k, w := range rl.windows {
				if now.After(w.resetAt) {
					delete(rl.windows, k)
				}
			}
			rl.mu.Unlock()
		}
	}
}
