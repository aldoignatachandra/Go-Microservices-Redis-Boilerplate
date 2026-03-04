// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
)

// MiddlewareRegistry provides access to common middleware with configuration.
type MiddlewareRegistry struct {
	config *config.Config
	logger logger.Logger
}

// NewMiddlewareRegistry creates a new middleware registry.
func NewMiddlewareRegistry(cfg *config.Config, log logger.Logger) *MiddlewareRegistry {
	return &MiddlewareRegistry{
		config: cfg,
		logger: log,
	}
}

// DefaultMiddleware returns all default middleware in the correct order.
func (r *MiddlewareRegistry) DefaultMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		r.RequestID(),
		r.Logger(),
		r.Recovery(),
		r.CORS(),
	}
}

// RequestID returns the request ID middleware.
func (r *MiddlewareRegistry) RequestID() gin.HandlerFunc {
	return RequestID()
}

// Logger returns the logging middleware.
func (r *MiddlewareRegistry) Logger() gin.HandlerFunc {
	return Logging(LoggingConfig{
		Logger:    r.logger,
		SkipPaths: []string{"/health", "/ready", "/live"},
	})
}

// Recovery returns the panic recovery middleware.
func (r *MiddlewareRegistry) Recovery() gin.HandlerFunc {
	return Recovery(RecoveryConfig{
		Logger:     r.logger,
		StackTrace: r.config.App.Env == "development",
	})
}

// CORS returns the CORS middleware.
func (r *MiddlewareRegistry) CORS() gin.HandlerFunc {
	config := DefaultCORSConfig()

	// Configure based on environment
	if r.config.App.Env == "production" {
		// In production, only allow specific origins
		config.AllowedOrigins = []string{"https://example.com"}
		config.AllowCredentials = true
	}

	return CORS(config)
}

// Auth returns the authentication middleware.
func (r *MiddlewareRegistry) Auth(skipPaths ...string) gin.HandlerFunc {
	return Auth(AuthConfig{
		JWTSecret: []byte(r.config.Auth.JWT.Secret),
		SkipPaths: skipPaths,
	})
}

// RateLimit returns the rate limiting middleware.
func (r *MiddlewareRegistry) RateLimit(requestsPerSecond float64, burst int) gin.HandlerFunc {
	return RateLimit(RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burst,
	})
}

// Timeout returns the timeout middleware.
func (r *MiddlewareRegistry) Timeout(timeout time.Duration) gin.HandlerFunc {
	return Timeout(TimeoutConfig{
		Timeout: timeout,
	})
}

// AdminOnly returns middleware that requires admin role.
func (r *MiddlewareRegistry) AdminOnly() gin.HandlerFunc {
	return RequireAdmin()
}

// PublicRoutes returns paths that should be publicly accessible.
func (r *MiddlewareRegistry) PublicRoutes() []string {
	return []string{
		"/health",
		"/ready",
		"/live",
		"/started",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
	}
}

// HealthCheckRoutes returns health check routes.
func (r *MiddlewareRegistry) HealthCheckRoutes() []string {
	return []string{
		"/health",
		"/ready",
		"/live",
		"/started",
	}
}
