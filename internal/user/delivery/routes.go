// Package delivery provides route registration for the user service.
package delivery

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all user service routes.
func RegisterRoutes(r *gin.Engine, handler *UserHandler) {
	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			// Public routes (with auth middleware in actual implementation)
			users.GET("/:id", handler.GetUser)
			users.GET("", handler.ListUsers)
			users.GET("/:id/profile", handler.GetProfile)

			// Protected routes (require auth)
			users.PUT("/profile", handler.UpdateProfile)

			// Admin routes
			users.POST("/:id/activate", handler.ActivateUser)
			users.POST("/:id/deactivate", handler.DeactivateUser)
			users.DELETE("/:id", handler.DeleteUser)
			users.POST("/:id/restore", handler.RestoreUser)
		}

		// Activity log routes
		v1.GET("/activity-logs", handler.GetActivityLogs)
	}
}
