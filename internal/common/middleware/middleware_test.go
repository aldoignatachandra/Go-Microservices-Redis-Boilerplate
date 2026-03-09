// Package middleware provides tests for common middleware utilities.
package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/internal/common/constants"
	"github.com/ignata/go-microservices-boilerplate/internal/common/middleware"
)

func TestPaginationParams_GetPage(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected int
	}{
		{"default when zero", 0, 1},
		{"default when negative", -1, 1},
		{"valid page 1", 1, 1},
		{"valid page 5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Page: tt.page}
			assert.Equal(t, tt.expected, p.GetPage())
		})
	}
}

func TestPaginationParams_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{"default when zero", 0, 20},
		{"default when negative", -1, 20},
		{"valid limit 10", 10, 10},
		{"valid limit 50", 50, 50},
		{"capped at max 100", 200, 100},
		{"exact max", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Limit: tt.limit}
			assert.Equal(t, tt.expected, p.GetLimit())
		})
	}
}

func TestPaginationParams_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		limit    int
		expected int
	}{
		{"page 1 limit 20", 1, 20, 0},
		{"page 2 limit 20", 2, 20, 20},
		{"page 3 limit 10", 3, 10, 20},
		{"page 1 limit 0 (default 20)", 1, 0, 0},
		{"page 0 (default 1) limit 20", 0, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Page: tt.page, Limit: tt.limit}
			assert.Equal(t, tt.expected, p.GetOffset())
		})
	}
}

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(c *gin.Context)
		expectOK bool
		expectID string
	}{
		{
			name:     "user ID exists",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserID), "user-123") },
			expectOK: true,
			expectID: "user-123",
		},
		{
			name:     "user ID not set",
			setupCtx: func(c *gin.Context) {},
			expectOK: false,
			expectID: "",
		},
		{
			name:     "user ID is empty string",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserID), "") },
			expectOK: false,
			expectID: "",
		},
		{
			name:     "user ID is wrong type",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserID), 123) },
			expectOK: false,
			expectID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			id, ok := middleware.ExtractUserID(c)
			assert.Equal(t, tt.expectOK, ok)
			assert.Equal(t, tt.expectID, id)
		})
	}
}

func TestExtractUserRole(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(c *gin.Context)
		expectOK   bool
		expectRole string
	}{
		{
			name:       "role exists as admin",
			setupCtx:   func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "ADMIN") },
			expectOK:   true,
			expectRole: "ADMIN",
		},
		{
			name:       "role exists as user",
			setupCtx:   func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "USER") },
			expectOK:   true,
			expectRole: "USER",
		},
		{
			name:       "role not set",
			setupCtx:   func(c *gin.Context) {},
			expectOK:   false,
			expectRole: "",
		},
		{
			name:       "role is empty string",
			setupCtx:   func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "") },
			expectOK:   false,
			expectRole: "",
		},
		{
			name:       "role is wrong type",
			setupCtx:   func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), 123) },
			expectOK:   false,
			expectRole: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			role, ok := middleware.ExtractUserRole(c)
			assert.Equal(t, tt.expectOK, ok)
			assert.Equal(t, tt.expectRole, role)
		})
	}
}

func TestExtractRequestID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(c *gin.Context)
		expected string
	}{
		{
			name:     "request ID exists",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyRequestID), "req-123") },
			expected: "req-123",
		},
		{
			name:     "request ID not set",
			setupCtx: func(c *gin.Context) {},
			expected: "",
		},
		{
			name:     "request ID is empty",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyRequestID), "") },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			requestID := middleware.ExtractRequestID(c)
			assert.Equal(t, tt.expected, requestID)
		})
	}
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(c *gin.Context)
		expected bool
	}{
		{
			name:     "is admin when role is ADMIN",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "ADMIN") },
			expected: true,
		},
		{
			name:     "is not admin when role is USER",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "USER") },
			expected: false,
		},
		{
			name:     "is not admin when role not set",
			setupCtx: func(c *gin.Context) {},
			expected: false,
		},
		{
			name:     "is not admin when role is empty",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), "") },
			expected: false,
		},
		{
			name:     "is not admin when role is wrong type",
			setupCtx: func(c *gin.Context) { c.Set(string(constants.ContextKeyUserRole), 123) },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			isAdmin := middleware.IsAdmin(c)
			assert.Equal(t, tt.expected, isAdmin)
		})
	}
}
