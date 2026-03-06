// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"context"
	"net/http"
	"sync"
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
	c.JSON(http.StatusRequestTimeout, gin.H{
		"error":   "request_timeout",
		"message": "Request processing timed out. Please try again.",
	})
}

// threadSafeWriter ensures thread-safe access to the ResponseWriter.
type threadSafeWriter struct {
	gin.ResponseWriter
	mu sync.Mutex
}

func (w *threadSafeWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ResponseWriter.Write(b)
}

func (w *threadSafeWriter) WriteHeader(statusCode int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *threadSafeWriter) WriteString(s string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ResponseWriter.WriteString(s)
}

// Timeout returns a middleware that adds timeout to requests.
func Timeout(config TimeoutConfig) gin.HandlerFunc {
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultTimeoutHandler
	}

	return func(c *gin.Context) {
		// Wrap the writer to make it thread-safe
		w := &threadSafeWriter{ResponseWriter: c.Writer}
		c.Writer = w

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
			w.mu.Lock()
			defer w.mu.Unlock()
			// If we haven't written status, assume 200 OK (Gin default)
			// But we don't need to force it if the handler didn't write anything
		case <-ctx.Done():
			// Timeout occurred
			w.mu.Lock()
			// We only allow the error handler to write if nothing has been written yet
			if !w.Written() {
				w.mu.Unlock() // Unlock before calling error handler which might write
				config.ErrorHandler(c)
				// We cannot safely call c.Abort() here because it modifies c.index, which is being read/modified
				// by c.Next() in the goroutine. This is a race condition.
				// By not calling Abort(), we allow the main chain to continue if this function returns,
				// but since we've already written the response, subsequent writes should be ignored/handled.
			} else {
				w.mu.Unlock()
			}
		}
	}
}

// TimeoutWithHandler returns a timeout middleware with custom error handler.
func TimeoutWithHandler(timeout time.Duration, handler func(*gin.Context)) gin.HandlerFunc {
	return Timeout(TimeoutConfig{
		Timeout:      timeout,
		ErrorHandler: handler,
	})
}

// RequestTimeout returns a simple timeout middleware with default error handler.
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return Timeout(TimeoutConfig{
		Timeout: timeout,
	})
}
