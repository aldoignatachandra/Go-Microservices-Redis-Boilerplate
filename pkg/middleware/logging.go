// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"go.uber.org/zap"
)

// LoggingConfig holds logging middleware configuration.
type LoggingConfig struct {
	// SkipPaths is a list of paths to skip logging
	SkipPaths []string

	// Logger is the logger to use
	Logger logger.Logger
}

// Logging returns a middleware that logs HTTP requests.
func Logging(config LoggingConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip logging for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID
		requestID := GetRequestID(c)

		// Get user ID if present
		userID, _ := c.Get("user_id")

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if userID != nil {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}

		// Log based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			config.Logger.Error("Server error", fields...)
		case status >= 400:
			config.Logger.Warn("Client error", fields...)
		default:
			config.Logger.Info("Request completed", fields...)
		}
	}
}

// ErrorLogging returns a middleware that logs errors from the context.
func ErrorLogging(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log any errors attached to the context
		if len(c.Errors) > 0 {
			requestID := GetRequestID(c)
			for _, e := range c.Errors {
				log.Error("Request error",
					zap.String("request_id", requestID),
					zap.String("error", e.Error()),
					zap.Int("type", int(e.Type)),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
			}
		}
	}
}
