// Package delivery provides route registration for the product service.
package delivery

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/middleware"
	"github.com/ignata/go-microservices-boilerplate/pkg/ratelimit"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
)

// CORSMiddleware provides CORS middleware for the product service.
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

// RegisterRoutes registers all product service routes.
func RegisterRoutes(
	r *gin.Engine,
	productUseCase usecase.ProductUseCase,
	jwtSecret string,
	sessionValidator middleware.SessionValidator,
) {
	handler := NewHandler(productUseCase)
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret:        []byte(jwtSecret),
		SessionValidator: sessionValidator,
	})

	products := r.Group("/products")
	products.Use(authMiddleware)
	products.GET("", handler.ListProducts)
	products.GET("/:id", handler.GetProduct)
	products.POST("", handler.CreateProduct)
	products.PUT("/:id", handler.UpdateProduct)
	products.DELETE("/:id", handler.DeleteProduct)
	products.POST("/:id/restore", handler.RestoreProduct)
	products.PUT("/:id/stock", handler.UpdateStock)
}

// RegisterRoutesWithRateLimit registers all product service routes with Redis-backed rate limiting.
func RegisterRoutesWithRateLimit(
	r *gin.Engine,
	productUseCase usecase.ProductUseCase,
	jwtSecret string,
	sessionValidator middleware.SessionValidator,
	redisLimiter *ratelimit.RouteRateLimiter,
	limit int,
	window time.Duration,
) {
	handler := NewHandler(productUseCase)
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret:        []byte(jwtSecret),
		SessionValidator: sessionValidator,
	})

	// Rate limiting middleware with per-route configuration
	rateLimitMiddleware := middleware.RedisRateLimitPerRoute(redisLimiter, limit, int(window.Seconds()))

	// Authenticated product routes with rate limiting
	products := r.Group("/products")
	products.Use(authMiddleware)
	products.Use(rateLimitMiddleware)
	products.GET("", handler.ListProducts)
	products.GET("/:id", handler.GetProduct)
	products.POST("", handler.CreateProduct)
	products.PUT("/:id", handler.UpdateProduct)
	products.DELETE("/:id", handler.DeleteProduct)
	products.POST("/:id/restore", handler.RestoreProduct)
	products.PUT("/:id/stock", handler.UpdateStock)
}

// PublicHealth is a public health check.
func (h *Handler) PublicHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "service-product",
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
