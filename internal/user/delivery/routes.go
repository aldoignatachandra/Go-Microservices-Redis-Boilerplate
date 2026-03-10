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
func RegisterRoutes(r *gin.Engine, handler *UserHandler, jwtSecret string, sessionValidator middleware.SessionValidator) {
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret:        []byte(jwtSecret),
		SessionValidator: sessionValidator,
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware)
	// User routes for authenticated user
	users := v1.Group("/users")
	users.GET("/:id", handler.GetUser)

	// Admin-only routes
	admin := v1.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.GET("/users", handler.ListUsers)
	admin.GET("/activity-logs", handler.GetActivityLogs)

	adminUsers := admin.Group("/users")
	adminUsers.POST("/:id/activate", handler.ActivateUser)
	adminUsers.POST("/:id/deactivate", handler.DeactivateUser)
	adminUsers.DELETE("/:id", handler.DeleteUser)
	adminUsers.POST("/:id/restore", handler.RestoreUser)
}

// RegisterRoutesWithRateLimit registers all user service routes with Redis-backed rate limiting.
func RegisterRoutesWithRateLimit(
	r *gin.Engine,
	handler *UserHandler,
	jwtSecret string,
	sessionValidator middleware.SessionValidator,
	redisLimiter *ratelimit.RouteRateLimiter,
	limit int,
	window time.Duration,
) {
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret:        []byte(jwtSecret),
		SessionValidator: sessionValidator,
	})
	rateLimitMiddleware := middleware.RedisRateLimitPerRoute(redisLimiter, limit, int(window.Seconds()))

	// API v1 group
	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware)
	// User self routes with rate limiting
	users := v1.Group("/users")
	users.Use(rateLimitMiddleware)
	users.GET("/:id", handler.GetUser)

	// Admin-only routes with rate limiting
	admin := v1.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.Use(rateLimitMiddleware)
	admin.GET("/users", handler.ListUsers)
	admin.GET("/activity-logs", handler.GetActivityLogs)

	adminUsers := admin.Group("/users")
	adminUsers.POST("/:id/activate", handler.ActivateUser)
	adminUsers.POST("/:id/deactivate", handler.DeactivateUser)
	adminUsers.DELETE("/:id", handler.DeleteUser)
	adminUsers.POST("/:id/restore", handler.RestoreUser)
}
