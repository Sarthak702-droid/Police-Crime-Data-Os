package middleware

import (
	"net/http"
	"sync"
	"time"

	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type clientLimiter struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	capacity float64
	rate     float64 // tokens per second
}

func NewRateLimiter(capacity float64, fillPeriod time.Duration) *RateLimiter {
	rate := capacity / fillPeriod.Seconds()
	return &RateLimiter{
		clients:  make(map[string]*clientLimiter),
		capacity: capacity,
		rate:     rate,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limiter, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientLimiter{
			tokens:     rl.capacity - 1.0,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(limiter.lastRefill).Seconds()
	limiter.tokens += elapsed * rl.rate
	if limiter.tokens > rl.capacity {
		limiter.tokens = rl.capacity
	}
	limiter.lastRefill = now

	if limiter.tokens >= 1.0 {
		limiter.tokens -= 1.0
		return true
	}

	return false
}

func RateLimitMiddleware(capacity float64, fillPeriod time.Duration, message string) gin.HandlerFunc {
	limiter := NewRateLimiter(capacity, fillPeriod)

	// Clean up old inactive clients in background
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			limiter.mu.Lock()
			for ip, cl := range limiter.clients {
				if time.Since(cl.lastRefill) > 30*time.Minute {
					delete(limiter.clients, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !limiter.Allow(clientIP) {
			utils.SendError(c, http.StatusTooManyRequests, message, "")
			c.Abort()
			return
		}
		c.Next()
	}
}
