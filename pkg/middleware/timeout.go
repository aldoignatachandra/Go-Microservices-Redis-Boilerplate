// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig holds timeout middleware configuration.
type TimeoutConfig struct {
	// Timeout is the maximum duration for handling a request
	Timeout time.Duration

	// ErrorHandler is a custom error handler for timeout.
	// If nil, default handler is used.
	ErrorHandler func(c *gin.Context)
}

// defaultTimeoutHandler is the default timeout error handler.
func defaultTimeoutHandler(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
		"error":   "request_timeout",
		"message": "Request processing timed out. Please try again.",
	})
}

// Timeout returns a middleware that adds timeout to requests.
func Timeout(config TimeoutConfig) gin.HandlerFunc {
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultTimeoutHandler
	}

	return func(c *gin.Context) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), config.Timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to signal completion
		finished := make(chan struct{})

		// Run handler in goroutine
		go func() {
			defer close(finished)
			c.Next()
		}()

		// Wait for completion or timeout
		select {
		case <-finished:
			// Request completed within timeout
			if !c.Writer.Written() {
				// If nothing was written, write status (may have been aborted)
				if c.Writer.Status() == 0 {
					c.Status(http.StatusOK)
				}
			}
		case <-ctx.Done():
			// Timeout occurred
			c.Abort()
			config.ErrorHandler(c)
		}
	}
}

// TimeoutWithHandler returns a timeout middleware with custom error handler.
func TimeoutWithHandler(timeout time.Duration, handler func(*gin.Context)) gin.HandlerFunc {
	return Timeout(TimeoutConfig{
		Timeout:     timeout,
		ErrorHandler: handler,
	})
}

// RequestTimeout returns a simple timeout middleware with default error handler.
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return Timeout(TimeoutConfig{
		Timeout: timeout,
	})
}
