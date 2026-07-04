// Package middleware holds cross-cutting Gin middleware (inbound rate limiting,
// request body caps) that isn't tied to auth or a specific handler.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// PerIPRateLimit token-bucket limits requests per client IP (gin's ClientIP,
// which honors SetTrustedProxies). Buckets idle >10m are evicted by a sweeper
// goroutine that lives for the process lifetime (one per middleware instance —
// construct once at startup, not per request group).
func PerIPRateLimit(rps float64, burst int) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		buckets = make(map[string]*ipBucket)
	)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, b := range buckets {
				if time.Since(b.lastSeen) > 10*time.Minute {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		b, ok := buckets[ip]
		if !ok {
			b = &ipBucket{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			buckets[ip] = b
		}
		b.lastSeen = time.Now()
		mu.Unlock()
		if !b.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Try again shortly.",
			})
			return
		}
		c.Next()
	}
}
