// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	// RequestsPerSecond is the number of requests allowed per second
	RequestsPerSecond float64

	// Burst is the maximum burst size
	Burst int

	// KeyFunc is a function to generate a unique key for each request.
	// Default: uses client IP address.
	KeyFunc func(*gin.Context) string

	// SkipFunc is a function to determine if a request should be skipped.
	SkipFunc func(*gin.Context) bool
}

// DefaultKeyFunc returns the client IP as the rate limit key.
func DefaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// IPRateLimiter is a rate limiter that tracks usage per IP.
type IPRateLimiter struct {
	limiters map[string]*limiter
	mu       sync.RWMutex
	config   RateLimiterConfig
}

type limiter struct {
	limiter *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter.
func NewIPRateLimiter(cfg RateLimiterConfig) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*limiter),
		config:   cfg,
	}
}

// getLimiter returns the rate limiter for the given key.
func (rl *IPRateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[key]
	if !exists {
		l := rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.Burst)
		rl.limiters[key] = &limiter{
			limiter:  l,
			lastSeen: time.Now(),
		}
		return l
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanupOldLimiters removes limiters that haven't been used recently.
func (rl *IPRateLimiter) cleanupOldLimiters(olderThan time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, entry := range rl.limiters {
		if time.Since(entry.lastSeen) > olderThan {
			delete(rl.limiters, key)
		}
	}
}

// RateLimit returns a rate limiting middleware.
func RateLimit(config RateLimiterConfig) gin.HandlerFunc {
	// Set defaults
	if config.KeyFunc == nil {
		config.KeyFunc = DefaultKeyFunc
	}

	limiter := NewIPRateLimiter(config)

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanupOldLimiters(10 * time.Minute)
		}
	}()

	return func(c *gin.Context) {
		// Skip if configured
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return
		}

		// Get rate limit key
		key := config.KeyFunc(c)

		// Check rate limit
		limiter := limiter.getLimiter(key)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}

// SlidingWindowRateLimiter implements a sliding window rate limiter.
type SlidingWindowRateLimiter struct {
	windows map[string][]time.Time
	mu      sync.RWMutex
	limit   int
	window  time.Duration
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter.
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	rl := &SlidingWindowRateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Allow checks if a request is allowed.
func (rl *SlidingWindowRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get or create window for key
	window := rl.windows[key]

	// Remove old timestamps
	var validTimestamps []time.Time
	for _, timestamp := range window {
		if timestamp.After(windowStart) {
			validTimestamps = append(validTimestamps, timestamp)
		}
	}

	// Check if limit is exceeded
	if len(validTimestamps) >= rl.limit {
		return false
	}

	// Add current timestamp
	validTimestamps = append(validTimestamps, now)
	rl.windows[key] = validTimestamps

	return true
}

// cleanup removes old entries.
func (rl *SlidingWindowRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	windowStart := time.Now().Add(-rl.window)

	for key, window := range rl.windows {
		var validTimestamps []time.Time
		for _, timestamp := range window {
			if timestamp.After(windowStart) {
				validTimestamps = append(validTimestamps, timestamp)
			}
		}

		if len(validTimestamps) == 0 {
			delete(rl.windows, key)
		} else {
			rl.windows[key] = validTimestamps
		}
	}
}

// SlidingWindowRateLimit returns a sliding window rate limiting middleware.
func SlidingWindowRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewSlidingWindowRateLimiter(limit, window)

	return func(c *gin.Context) {
		key := DefaultKeyFunc(c)

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d per %v.", limit, window),
			})
			return
		}

		c.Next()
	}
}
