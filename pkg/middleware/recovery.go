// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"go.uber.org/zap"
)

// RecoveryConfig holds recovery middleware configuration.
type RecoveryConfig struct {
	// Logger is the logger to use for logging panics
	Logger logger.Logger

	// StackTrace determines if stack trace should be logged
	StackTrace bool

	// ErrorHandler is a custom error handler. If nil, default handler is used.
	ErrorHandler func(c *gin.Context, err interface{})
}

// defaultErrorHandler is the default error handler for panics.
func defaultErrorHandler(c *gin.Context, err interface{}) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_server_error",
		"message": "An unexpected error occurred. Please try again later.",
	})
}

// Recovery returns a middleware that recovers from panics.
func Recovery(config RecoveryConfig) gin.HandlerFunc {
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				requestID := GetRequestID(c)
				fields := []zap.Field{
					zap.String("request_id", requestID),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("client_ip", c.ClientIP()),
					zap.Any("panic", err),
				}

				if config.StackTrace {
					fields = append(fields, zap.String("stack", string(debug.Stack())))
				}

				config.Logger.Error("Panic recovered", fields...)

				// Call error handler
				config.ErrorHandler(c, err)
			}
		}()

		c.Next()
	}
}

// CustomRecovery returns a recovery middleware with custom error response.
func CustomRecovery(log logger.Logger, customHandler func(*gin.Context, interface{})) gin.HandlerFunc {
	return Recovery(RecoveryConfig{
		Logger:      log,
		StackTrace:  true,
		ErrorHandler: customHandler,
	})
}

// RecoveryWithWriter returns a recovery middleware that writes to a custom writer.
func RecoveryWithWriter(log logger.Logger) gin.HandlerFunc {
	return Recovery(RecoveryConfig{
		Logger:     log,
		StackTrace: true,
	})
}

// PanicHandler creates a custom panic handler with detailed error info.
func PanicHandler(detail bool) func(c *gin.Context, err interface{}) {
	return func(c *gin.Context, err interface{}) {
		response := gin.H{
			"error":   "internal_server_error",
			"message": "An unexpected error occurred. Please try again later.",
		}

		// Add details in development mode
		if detail {
			response["panic"] = fmt.Sprintf("%v", err)
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, response)
	}
}
