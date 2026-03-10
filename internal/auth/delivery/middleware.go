// Package delivery provides middleware for the auth service.
package delivery

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// SessionValidator validates that a user still has an active server-side session.
type SessionValidator func(ctx context.Context, userID, sessionID string) (bool, error)

// AuthMiddleware validates JWT tokens and sets user info in context.
func AuthMiddleware(jwtSecret string, sessionValidator ...SessionValidator) gin.HandlerFunc {
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret: jwtSecret,
	})

	var validator SessionValidator
	if len(sessionValidator) > 0 {
		validator = sessionValidator[0]
	}

	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		// Extract token
		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			utils.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			utils.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		if validator != nil {
			valid, sessionErr := validator(c.Request.Context(), claims.UserID, claims.SessionID)
			if sessionErr != nil {
				utils.InternalError(c, "Failed to validate session")
				c.Abort()
				return
			}
			if !valid {
				utils.Unauthorized(c, "Invalid or expired token")
				c.Abort()
				return
			}
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminOnlyMiddleware requires admin role.
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.Unauthorized(c, "")
			c.Abort()
			return
		}

		if role.(string) != string(domain.RoleAdmin) {
			utils.Forbidden(c, utils.AdminAccessRequiredMessage)
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuthMiddleware optionally validates JWT tokens.
// It sets user info if token is valid, but doesn't require it.
func OptionalAuthMiddleware(jwtSecret string, sessionValidator ...SessionValidator) gin.HandlerFunc {
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret: jwtSecret,
	})

	var validator SessionValidator
	if len(sessionValidator) > 0 {
		validator = sessionValidator[0]
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		if validator != nil {
			valid, sessionErr := validator(c.Request.Context(), claims.UserID, claims.SessionID)
			if sessionErr != nil || !valid {
				c.Next()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// CORSMiddleware handles CORS.
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

// RateLimitKeyFunc returns a key for rate limiting.
func RateLimitKeyFunc(c *gin.Context) string {
	// Use user ID if authenticated, otherwise use IP
	userID, exists := c.Get("user_id")
	if exists {
		return "user:" + userID.(string)
	}
	return "ip:" + c.ClientIP()
}

// GetCurrentUserID extracts user ID from context.
func GetCurrentUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetCurrentUserEmail extracts email from context.
func GetCurrentUserEmail(c *gin.Context) string {
	email, exists := c.Get("email")
	if !exists {
		return ""
	}
	return email.(string)
}

// GetCurrentUserRole extracts role from context.
func GetCurrentUserRole(c *gin.Context) domain.Role {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	return domain.Role(role.(string))
}

// IsAuthenticated checks if the request is authenticated.
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}

// IsAdmin checks if the current user is an admin.
func IsAdmin(c *gin.Context) bool {
	role := GetCurrentUserRole(c)
	return role == domain.RoleAdmin
}

// HasRole checks if the current user has a specific role.
func HasRole(c *gin.Context, requiredRole domain.Role) bool {
	role := GetCurrentUserRole(c)
	return role == requiredRole
}

// RequireRoles requires one of the specified roles.
func RequireRoles(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := GetCurrentUserRole(c)
		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}
		utils.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}
