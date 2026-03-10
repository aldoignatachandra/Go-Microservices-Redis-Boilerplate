// Package delivery tests middleware for the auth service.
package delivery_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// TestAuthMiddleware_Success tests successful authentication with valid token.
func TestAuthMiddleware_Success(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key-for-testing"
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    jwtSecret,
		ExpiresIn: 3600 * time.Second, // 1 hour
	})

	token, err := jwtManager.GenerateToken("550e8400-e29b-41d4-a716-446655440001", "test@example.com", string(domain.RoleUser))
	require.NoError(t, err)

	router.Use(delivery.AuthMiddleware(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_SessionBoundToken ensures validator receives token session ID.
func TestAuthMiddleware_SessionBoundToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key-for-testing"
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    jwtSecret,
		ExpiresIn: 3600 * time.Second,
	})

	const expectedUserID = "550e8400-e29b-41d4-a716-446655440001"
	const expectedSessionID = "6e0f7d89-32ff-4d58-a005-fce95d6b8e45"
	token, err := jwtManager.GenerateTokenWithSession(expectedUserID, "test@example.com", string(domain.RoleUser), expectedSessionID)
	require.NoError(t, err)

	var gotUserID, gotSessionID string
	router.Use(delivery.AuthMiddleware(jwtSecret, func(_ context.Context, userID, sessionID string) (bool, error) {
		gotUserID = userID
		gotSessionID = sessionID
		return true, nil
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expectedUserID, gotUserID)
	assert.Equal(t, expectedSessionID, gotSessionID)
}

// TestAuthMiddleware_MissingHeader tests authentication with missing Authorization header.
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key"
	router.Use(delivery.AuthMiddleware(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_InvalidFormat tests authentication with invalid header format.
func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing Bearer prefix",
			header: "invalid-token",
		},
		{
			name:   "wrong prefix",
			header: "Basic token123",
		},
		{
			name:   "only Bearer",
			header: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			router := gin.New()

			jwtSecret := "test-secret-key"
			router.Use(delivery.AuthMiddleware(jwtSecret))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "success"})
			})

			// Act
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestAuthMiddleware_InvalidToken tests authentication with invalid token.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key"
	router.Use(delivery.AuthMiddleware(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RevokedSession tests authentication with revoked server-side session.
func TestAuthMiddleware_RevokedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key-for-testing"
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    jwtSecret,
		ExpiresIn: 3600 * time.Second,
	})

	token, err := jwtManager.GenerateToken("550e8400-e29b-41d4-a716-446655440001", "test@example.com", string(domain.RoleUser))
	require.NoError(t, err)

	router.Use(delivery.AuthMiddleware(jwtSecret, func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_SessionValidatorError tests middleware behavior on validator failure.
func TestAuthMiddleware_SessionValidatorError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key-for-testing"
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    jwtSecret,
		ExpiresIn: 3600 * time.Second,
	})

	token, err := jwtManager.GenerateToken("550e8400-e29b-41d4-a716-446655440001", "test@example.com", string(domain.RoleUser))
	require.NoError(t, err)

	router.Use(delivery.AuthMiddleware(jwtSecret, func(_ context.Context, _, _ string) (bool, error) {
		return false, errors.New("db unavailable")
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAdminOnlyMiddleware_Success tests admin middleware with admin user.
func TestAdminOnlyMiddleware_Success(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-123")
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	})

	router.Use(delivery.AdminOnlyMiddleware())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "admin access"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminOnlyMiddleware_NotAdmin tests admin middleware with non-admin user.
func TestAdminOnlyMiddleware_NotAdmin(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("role", string(domain.RoleUser))
		c.Next()
	})

	router.Use(delivery.AdminOnlyMiddleware())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "admin access"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestAdminOnlyMiddleware_NoRole tests admin middleware with no role set.
func TestAdminOnlyMiddleware_NoRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Don't set any role
	router.Use(delivery.AdminOnlyMiddleware())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "admin access"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestOptionalAuthMiddleware_WithToken tests optional auth with valid token.
func TestOptionalAuthMiddleware_WithToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key-for-testing"
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    jwtSecret,
		ExpiresIn: 3600 * time.Second,
	})

	token, err := jwtManager.GenerateToken("550e8400-e29b-41d4-a716-446655440001", "test@example.com", string(domain.RoleUser))
	require.NoError(t, err)

	router.Use(delivery.OptionalAuthMiddleware(jwtSecret))
	router.GET("/optional", func(c *gin.Context) {
		userID := delivery.GetCurrentUserID(c)
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestOptionalAuthMiddleware_WithoutToken tests optional auth without token.
func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key"
	router.Use(delivery.OptionalAuthMiddleware(jwtSecret))
	router.GET("/optional", func(c *gin.Context) {
		userID := delivery.GetCurrentUserID(c)
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/optional", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestOptionalAuthMiddleware_InvalidToken tests optional auth with invalid token.
func TestOptionalAuthMiddleware_InvalidToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := "test-secret-key"
	router.Use(delivery.OptionalAuthMiddleware(jwtSecret))
	router.GET("/optional", func(c *gin.Context) {
		userID := delivery.GetCurrentUserID(c)
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert - Should still pass through, just without user info
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCORSMiddleware_Options tests CORS middleware with OPTIONS request.
func TestCORSMiddleware_Options(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(delivery.CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORSMiddleware_Get tests CORS middleware with GET request.
func TestCORSMiddleware_Get(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(delivery.CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST, OPTIONS, GET, PUT, DELETE, PATCH", w.Header().Get("Access-Control-Allow-Methods"))
}

// TestRateLimitKeyFunc_Authenticated tests rate limit key for authenticated user.
func TestRateLimitKeyFunc_Authenticated(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		key := delivery.RateLimitKeyFunc(c)
		c.JSON(200, gin.H{"key": key})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRateLimitKeyFunc_Unauthenticated tests rate limit key for unauthenticated user.
func TestRateLimitKeyFunc_Unauthenticated(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		key := delivery.RateLimitKeyFunc(c)
		c.JSON(200, gin.H{"key": key})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetCurrentUserID tests getting current user ID.
func TestGetCurrentUserID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		userID := delivery.GetCurrentUserID(c)
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetCurrentUserID_NotSet tests getting user ID when not set.
func TestGetCurrentUserID_NotSet(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		userID := delivery.GetCurrentUserID(c)
		c.JSON(200, gin.H{"user_id": userID})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetCurrentUserEmail tests getting current user email.
func TestGetCurrentUserEmail(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("email", "test@example.com")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		email := delivery.GetCurrentUserEmail(c)
		c.JSON(200, gin.H{"email": email})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetCurrentUserRole tests getting current user role.
func TestGetCurrentUserRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		role := delivery.GetCurrentUserRole(c)
		c.JSON(200, gin.H{"role": role})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestIsAuthenticated tests checking if user is authenticated.
func TestIsAuthenticated(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		isAuth := delivery.IsAuthenticated(c)
		c.JSON(200, gin.H{"authenticated": isAuth})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestIsAuthenticated_NotAuthenticated tests checking authentication when not authenticated.
func TestIsAuthenticated_NotAuthenticated(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		isAuth := delivery.IsAuthenticated(c)
		c.JSON(200, gin.H{"authenticated": isAuth})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestIsAdmin tests checking if user is admin.
func TestIsAdmin(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		isAdmin := delivery.IsAdmin(c)
		c.JSON(200, gin.H{"is_admin": isAdmin})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHasRole tests checking if user has specific role.
func TestHasRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		hasRole := delivery.HasRole(c, domain.RoleAdmin)
		c.JSON(200, gin.H{"has_role": hasRole})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRequireRoles_Success tests requiring specific role.
func TestRequireRoles_Success(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", string(domain.RoleAdmin))
		c.Next()
	})

	router.Use(delivery.RequireRoles(domain.RoleAdmin, domain.RoleUser))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRequireRoles_InsufficientPermissions tests requiring role with insufficient permissions.
func TestRequireRoles_InsufficientPermissions(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("role", string(domain.RoleUser))
		c.Next()
	})

	router.Use(delivery.RequireRoles(domain.RoleAdmin))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestRequireRoles_NoRole tests requiring role when no role is set.
func TestRequireRoles_NoRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(delivery.RequireRoles(domain.RoleAdmin))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
}
