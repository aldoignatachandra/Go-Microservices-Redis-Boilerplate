// Package middleware provides common HTTP middleware for Go microservices.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

const (
	// User ID key in context
	UserIDKey = "user_id"
	// User role key in context
	UserRoleKey = "user_role"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// JWTSecret is the secret key for JWT validation
	JWTSecret []byte

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string
}

// Auth returns an authentication middleware using JWT.
func Auth(config AuthConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip authentication for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required", nil)
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header format", nil)
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := utils.ValidateJWT(token, config.JWTSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole returns a middleware that requires specific roles.
func RequireRole(roles ...domain.Role) gin.HandlerFunc {
	allowedRoles := make(map[domain.Role]bool)
	for _, role := range roles {
		allowedRoles[role] = true
	}

	return func(c *gin.Context) {
		// Get user role from context
		userRole, exists := c.Get(UserRoleKey)
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required", nil)
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user role in context", nil)
			c.Abort()
			return
		}

		// Check if role is allowed
		if !allowedRoles[domain.Role(roleStr)] {
			utils.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(domain.RoleAdmin)
}

// OptionalAuth is an authentication middleware that doesn't require auth
// but will set user info if a valid token is provided.
func OptionalAuth(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]

		// Try to validate token
		claims, err := utils.ValidateJWT(token, config.JWTSecret)
		if err == nil {
			// Set user info in context if token is valid
			c.Set(UserIDKey, claims.UserID)
			c.Set(UserRoleKey, claims.Role)
		}

		c.Next()
	}
}

// GetUserID retrieves the user ID from the Gin context.
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUserRole retrieves the user role from the Gin context.
func GetUserRole(c *gin.Context) (domain.Role, bool) {
	userRole, exists := c.Get(UserRoleKey)
	if !exists {
		return "", false
	}
	role, ok := userRole.(string)
	return domain.Role(role), ok
}

// IsAuthenticated checks if the request is authenticated.
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(UserIDKey)
	return exists
}

// IsAdmin checks if the user has admin role.
func IsAdmin(c *gin.Context) bool {
	role, exists := GetUserRole(c)
	return exists && role == domain.RoleAdmin
}
