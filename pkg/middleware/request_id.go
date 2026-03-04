// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey = "request_id"
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"
)

// RequestID adds a unique request ID to each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get existing request ID from header
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new one if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context
		c.Set(RequestIDKey, requestID)

		// Set in response header
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the Gin context.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
