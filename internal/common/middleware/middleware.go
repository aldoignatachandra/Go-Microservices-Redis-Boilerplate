// Package middleware provides shared middleware utilities across all microservices.
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/common/constants"
)

// ExtractUserID extracts the user ID from the Gin context (set by auth middleware).
func ExtractUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(string(constants.ContextKeyUserID))
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok && id != ""
}

// ExtractUserRole extracts the user role from the Gin context.
func ExtractUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(string(constants.ContextKeyUserRole))
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok && r != ""
}

// ExtractRequestID extracts the request ID from the Gin context.
func ExtractRequestID(c *gin.Context) string {
	return c.GetString(string(constants.ContextKeyRequestID))
}

// IsAdmin checks if the current user has the admin role.
func IsAdmin(c *gin.Context) bool {
	role, ok := ExtractUserRole(c)
	return ok && role == constants.RoleAdmin
}

// PaginationParams represents pagination query parameters.
type PaginationParams struct {
	Page  int `form:"page" binding:"omitempty,min=1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// GetPage returns the page number, defaulting to 1.
func (p *PaginationParams) GetPage() int {
	if p.Page <= 0 {
		return constants.DefaultPage
	}
	return p.Page
}

// GetLimit returns the limit, defaulting to 20.
func (p *PaginationParams) GetLimit() int {
	if p.Limit <= 0 {
		return constants.DefaultLimit
	}
	if p.Limit > constants.MaxLimit {
		return constants.MaxLimit
	}
	return p.Limit
}

// GetOffset calculates the offset for database queries.
func (p *PaginationParams) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}
