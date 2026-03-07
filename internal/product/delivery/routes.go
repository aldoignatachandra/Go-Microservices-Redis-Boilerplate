// Package delivery provides route registration for the product service.
package delivery

import (
	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
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
func RegisterRoutes(r *gin.Engine, productUseCase usecase.ProductUseCase) {
	handler := NewHandler(productUseCase)

	// Health check endpoints (public)
	r.GET("/health", handler.PublicHealth)
	r.GET("/ready", handler.ReadyProbe)
	r.GET("/live", handler.LiveProbe)

	// Public product routes
	r.GET("/products", handler.ListProducts)
	r.GET("/products/:id", handler.GetProduct)

	// Protected product routes (require authentication)
	r.POST("/products", handler.CreateProduct)
	r.PUT("/products/:id", handler.UpdateProduct)
	r.DELETE("/products/:id", handler.DeleteProduct)
	r.POST("/products/:id/restore", handler.RestoreProduct)
	r.PUT("/products/:id/stock", handler.UpdateStock)
}

// PublicHealth is a public health check.
func (h *Handler) PublicHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "product-service",
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
