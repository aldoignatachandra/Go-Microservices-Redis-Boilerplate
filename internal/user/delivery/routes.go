// Package delivery provides route registration for the user service.
package delivery

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/pkg/middleware"
	"github.com/ignata/go-microservices-boilerplate/pkg/ratelimit"
)

// CORSMiddleware provides CORS middleware for the user service.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RegisterRoutes registers all user service routes.
func RegisterRoutes(r *gin.Engine, handler *UserHandler) {
	// API v1 group
	v1 := r.Group("/api/v1")
	// User routes
	users := v1.Group("/users")
	// Public routes (with auth middleware in actual implementation)
	users.GET("/:id", handler.GetUser)
	users.GET("", handler.ListUsers)
	// Admin routes
	users.POST("/:id/activate", handler.ActivateUser)
	users.POST("/:id/deactivate", handler.DeactivateUser)
	users.DELETE("/:id", handler.DeleteUser)
	users.POST("/:id/restore", handler.RestoreUser)
	// Activity log routes
	v1.GET("/activity-logs", handler.GetActivityLogs)
}

// RegisterRoutesWithRateLimit registers all user service routes with Redis-backed rate limiting.
func RegisterRoutesWithRateLimit(
	r *gin.Engine,
	handler *UserHandler,
	redisLimiter *ratelimit.RouteRateLimiter,
	limit int,
	window time.Duration,
) {
	rateLimitMiddleware := middleware.RedisRateLimitPerRoute(redisLimiter, limit, int(window.Seconds()))

	// API v1 group
	v1 := r.Group("/api/v1")
	// User routes with rate limiting
	users := v1.Group("/users")
	users.Use(rateLimitMiddleware)
	// Public routes
	users.GET("/:id", handler.GetUser)
	users.GET("", handler.ListUsers)
	// Admin routes
	users.POST("/:id/activate", handler.ActivateUser)
	users.POST("/:id/deactivate", handler.DeactivateUser)
	users.DELETE("/:id", handler.DeleteUser)
	users.POST("/:id/restore", handler.RestoreUser)
	// Activity log routes
	v1.GET("/activity-logs", handler.GetActivityLogs)
}
