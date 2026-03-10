// Package delivery provides route registration for the auth service.
package delivery

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/middleware"
	"github.com/ignata/go-microservices-boilerplate/pkg/ratelimit"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
)

// RegisterRoutes registers all auth service routes.
func RegisterRoutes(r *gin.Engine, authUseCase usecase.AuthUseCase, jwtSecret string) {
	handler := NewHandler(authUseCase)

	// Public auth routes
	auth := r.Group("/auth")
	auth.POST("/register", handler.Register)
	auth.POST("/login", handler.Login)
	auth.POST("/refresh", handler.RefreshToken)

	// Protected auth routes (require authentication)
	authProtected := r.Group("/auth")
	authProtected.Use(AuthMiddleware(jwtSecret))
	authProtected.POST("/logout", handler.Logout)
	authProtected.GET("/me", handler.GetCurrentUser)
	authProtected.POST("/change-password", handler.ChangePassword)

	// Admin routes (require admin role)
	admin := r.Group("/admin")
	admin.Use(AuthMiddleware(jwtSecret))
	admin.Use(AdminOnlyMiddleware())
	admin.GET("/users", handler.ListUsers)
	admin.GET("/users/:id", handler.GetUser)
	admin.DELETE("/users/:id", handler.DeleteUser)
	admin.POST("/users/:id/restore", handler.RestoreUser)
}

// RegisterRoutesWithRateLimit registers all auth service routes with Redis-backed rate limiting.
func RegisterRoutesWithRateLimit(
	r *gin.Engine,
	authUseCase usecase.AuthUseCase,
	jwtSecret string,
	redisLimiter *ratelimit.RouteRateLimiter,
	limit int,
	window time.Duration,
) {
	handler := NewHandler(authUseCase)

	// Rate limiting middleware with per-route configuration
	rateLimitMiddleware := middleware.RedisRateLimitPerRoute(redisLimiter, limit, int(window.Seconds()))

	// Public auth routes with rate limiting
	auth := r.Group("/auth")
	auth.Use(rateLimitMiddleware)
	auth.POST("/register", handler.Register)
	auth.POST("/login", handler.Login)
	auth.POST("/refresh", handler.RefreshToken)

	// Protected auth routes (require authentication)
	authProtected := r.Group("/auth")
	authProtected.Use(AuthMiddleware(jwtSecret))
	authProtected.POST("/logout", handler.Logout)
	authProtected.GET("/me", handler.GetCurrentUser)
	authProtected.POST("/change-password", handler.ChangePassword)

	// Admin routes (require admin role)
	admin := r.Group("/admin")
	admin.Use(AuthMiddleware(jwtSecret))
	admin.Use(AdminOnlyMiddleware())
	admin.GET("/users", handler.ListUsers)
	admin.GET("/users/:id", handler.GetUser)
	admin.DELETE("/users/:id", handler.DeleteUser)
	admin.POST("/users/:id/restore", handler.RestoreUser)
}

// PublicHealth is a public health check.
func (h *Handler) PublicHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "service-auth",
	})
}

// ReadyProbe is a readiness probe.
func (h *Handler) ReadyProbe(c *gin.Context) {
	c.JSON(200, gin.H{
		"ready": true,
	})
}

// LiveProbe is a liveness probe.
func (h *Handler) LiveProbe(c *gin.Context) {
	c.JSON(200, gin.H{
		"alive": true,
	})
}

// RegisterHealthRoutes registers health check routes.
func RegisterHealthRoutes(r *gin.Engine, healthHandler *server.HealthHandler) {
	// Public health routes
	r.GET("/health", healthHandler.PublicHealth)
	r.GET("/ready", healthHandler.ReadyProbe)
	r.GET("/live", healthHandler.LiveProbe)

	// Admin health routes (should be protected)
	admin := r.Group("/admin")
	admin.GET("/health", healthHandler.AdminHealth)
}
