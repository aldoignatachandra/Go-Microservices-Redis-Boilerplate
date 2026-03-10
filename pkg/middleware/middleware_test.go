// Package middleware provides tests for common HTTP middleware.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAuth tests the authentication middleware.
func TestAuth(t *testing.T) {
	// Setup
	jwtSecret := []byte("test-secret-key-for-auth-middleware-tests")

	tests := []struct {
		name           string
		config         AuthConfig
		setupRequest   func(*http.Request)
		expectedStatus int
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name: "valid JWT token",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
			},
			setupRequest: func(req *http.Request) {
				// Generate a valid JWT token using JWTManager
				manager := utils.NewJWTManager(utils.JWTConfig{
					Secret:    string(jwtSecret),
					ExpiresIn: time.Hour,
				})
				token, err := manager.GenerateToken("user-123", "test@example.com", string(domain.RoleUser))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				userID, exists := GetUserID(c)
				assert.True(t, exists, "user ID should exist in context")
				assert.Equal(t, "user-123", userID)
			},
		},
		{
			name: "revoked session",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
				SessionValidator: func(_ context.Context, _, _ string) (bool, error) {
					return false, nil
				},
			},
			setupRequest: func(req *http.Request) {
				manager := utils.NewJWTManager(utils.JWTConfig{
					Secret:    string(jwtSecret),
					ExpiresIn: time.Hour,
				})
				token, err := manager.GenerateToken("user-123", "test@example.com", string(domain.RoleUser))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "session validator error",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
				SessionValidator: func(_ context.Context, _, _ string) (bool, error) {
					return false, errors.New("db unavailable")
				},
			},
			setupRequest: func(req *http.Request) {
				manager := utils.NewJWTManager(utils.JWTConfig{
					Secret:    string(jwtSecret),
					ExpiresIn: time.Hour,
				})
				token, err := manager.GenerateToken("user-123", "test@example.com", string(domain.RoleUser))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "missing authorization header",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
			},
			setupRequest: func(req *http.Request) {
				// No authorization header
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid authorization header format",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
			},
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidFormat token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
			},
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "skip path",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{"/health", "/public"},
			},
			setupRequest: func(req *http.Request) {
				// No authorization header but path is in skip list
				// Update the request URL to match a skipped path
				req.URL.Path = "/health"
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired token",
			config: AuthConfig{
				JWTSecret: jwtSecret,
				SkipPaths: []string{},
			},
			setupRequest: func(req *http.Request) {
				// Generate an expired token
				manager := utils.NewJWTManager(utils.JWTConfig{
					Secret:    string(jwtSecret),
					ExpiresIn: -time.Hour, // Negative duration = expired
				})
				token, err := manager.GenerateToken("user-123", "test@example.com", string(domain.RoleUser))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(Auth(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.GET("/health", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create base request - path will be modified by setupRequest
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAuth_SessionBoundToken(t *testing.T) {
	jwtSecret := []byte("test-secret-key-for-auth-middleware-tests")
	manager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    string(jwtSecret),
		ExpiresIn: time.Hour,
	})

	const expectedUserID = "user-123"
	const expectedSessionID = "5ff70a4d-3a43-49e8-a2e3-92682e5acc44"
	token, err := manager.GenerateTokenWithSession(expectedUserID, "test@example.com", string(domain.RoleUser), expectedSessionID)
	require.NoError(t, err)

	var gotUserID, gotSessionID string
	router := gin.New()
	router.Use(Auth(AuthConfig{
		JWTSecret: jwtSecret,
		SessionValidator: func(_ context.Context, userID, sessionID string) (bool, error) {
			gotUserID = userID
			gotSessionID = sessionID
			return true, nil
		},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expectedUserID, gotUserID)
	assert.Equal(t, expectedSessionID, gotSessionID)
}

// TestRequireRole tests the role-based authorization middleware.
func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		roles          []domain.Role
		setupContext   func(*gin.Context)
		expectedStatus int
	}{
		{
			name:  "user has required role",
			roles: []domain.Role{domain.RoleAdmin},
			setupContext: func(c *gin.Context) {
				c.Set(UserIDKey, "user-123")
				c.Set(UserRoleKey, string(domain.RoleAdmin))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "user lacks required role",
			roles: []domain.Role{domain.RoleAdmin},
			setupContext: func(c *gin.Context) {
				c.Set(UserIDKey, "user-123")
				c.Set(UserRoleKey, string(domain.RoleUser))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:  "user not authenticated",
			roles: []domain.Role{domain.RoleAdmin},
			setupContext: func(c *gin.Context) {
				// No user info set
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:  "multiple allowed roles",
			roles: []domain.Role{domain.RoleAdmin, domain.RoleUser},
			setupContext: func(c *gin.Context) {
				c.Set(UserIDKey, "user-123")
				c.Set(UserRoleKey, string(domain.RoleUser))
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(func(c *gin.Context) {
				tt.setupContext(c)
				c.Next()
			})
			router.Use(RequireRole(tt.roles...))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestRequireAdmin tests the admin-only middleware.
func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name           string
		userRole       string
		expectedStatus int
	}{
		{
			name:           "admin user",
			userRole:       string(domain.RoleAdmin),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "regular user",
			userRole:       string(domain.RoleUser),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no role",
			userRole:       "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.userRole != "" {
					c.Set(UserRoleKey, tt.userRole)
				}
				c.Next()
			})
			router.Use(RequireAdmin())
			router.GET("/admin", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/admin", nil)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRequireAdmin_ForbiddenMessageConsistency(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserRoleKey, string(domain.RoleUser))
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, utils.AdminAccessRequiredMessage, body["message"])
	if dataObj, ok := body["data"].(map[string]interface{}); ok {
		assert.Equal(t, "FORBIDDEN", dataObj["code"])
		assert.Equal(t, utils.AdminAccessRequiredMessage, dataObj["message"])
	} else {
		t.Fatalf("expected data object in forbidden response")
	}
}

// TestOptionalAuth tests the optional authentication middleware.
func TestOptionalAuth(t *testing.T) {
	jwtSecret := []byte("test-secret-key-for-optional-auth")

	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		checkContext func(*testing.T, *gin.Context)
	}{
		{
			name: "valid token - user info set",
			setupRequest: func(req *http.Request) {
				manager := utils.NewJWTManager(utils.JWTConfig{
					Secret:    string(jwtSecret),
					ExpiresIn: time.Hour,
				})
				token, err := manager.GenerateToken("user-123", "test@example.com", string(domain.RoleUser))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
			},
			checkContext: func(t *testing.T, c *gin.Context) {
				userID, exists := GetUserID(c)
				assert.True(t, exists, "user ID should exist")
				assert.Equal(t, "user-123", userID)
			},
		},
		{
			name: "no token - request continues",
			setupRequest: func(req *http.Request) {
				// No authorization header
			},
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := GetUserID(c)
				assert.False(t, exists, "user ID should not exist")
			},
		},
		{
			name: "invalid token - request continues",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := GetUserID(c)
				assert.False(t, exists, "user ID should not exist")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(OptionalAuth(AuthConfig{JWTSecret: jwtSecret}))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestRateLimit tests the rate limiting middleware.
func TestRateLimit(t *testing.T) {
	tests := []struct {
		name           string
		config         RateLimiterConfig
		makeRequests   func(int) []*http.Request
		expectedStatus int
		requestCount   int
	}{
		{
			name: "allow requests under limit",
			config: RateLimiterConfig{
				RequestsPerSecond: 1,
				Burst:             1,
			},
			makeRequests: func(count int) []*http.Request {
				reqs := make([]*http.Request, count)
				for i := 0; i < count; i++ {
					reqs[i] = httptest.NewRequest("GET", "/test", nil)
				}
				return reqs
			},
			requestCount:   1,
			expectedStatus: http.StatusOK,
		},
		{
			name: "block requests over limit",
			config: RateLimiterConfig{
				RequestsPerSecond: 0.1, // 1 request per 10 seconds
				Burst:             1,
			},
			makeRequests: func(count int) []*http.Request {
				reqs := make([]*http.Request, count)
				for i := 0; i < count; i++ {
					reqs[i] = httptest.NewRequest("GET", "/test", nil)
				}
				return reqs
			},
			requestCount:   2, // First allowed, second blocked
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name: "skip function",
			config: RateLimiterConfig{
				RequestsPerSecond: 1,
				Burst:             1,
				SkipFunc: func(c *gin.Context) bool {
					return c.Request.Header.Get("X-Skip-RateLimit") == "true"
				},
			},
			makeRequests: func(count int) []*http.Request {
				reqs := make([]*http.Request, count)
				for i := 0; i < count; i++ {
					req := httptest.NewRequest("GET", "/test", nil)
					req.Header.Set("X-Skip-RateLimit", "true")
					reqs[i] = req
				}
				return reqs
			},
			requestCount:   5,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(RateLimit(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			requests := tt.makeRequests(tt.requestCount)

			// Act
			var lastStatus int
			for _, req := range requests {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				lastStatus = w.Code
			}

			// Assert - Check the last request's status
			if tt.name == "block requests over limit" {
				// First request should succeed, second should be rate limited
				assert.Equal(t, http.StatusTooManyRequests, lastStatus)
			} else {
				assert.Equal(t, tt.expectedStatus, lastStatus)
			}
		})
	}
}

func TestDefaultRedisKeyFunc_PrefersUserID(t *testing.T) {
	router := gin.New()
	var gotKey string

	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, "user-123")
		c.Next()
	})
	router.GET("/api/v1/users", func(c *gin.Context) {
		gotKey = defaultRedisKeyFunc(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.RemoteAddr = "198.51.100.8:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user:user-123:GET:/api/v1/users", gotKey)
}

func TestDefaultRedisKeyFunc_FallsBackToIP(t *testing.T) {
	router := gin.New()
	var gotKey string

	router.GET("/api/v1/users", func(c *gin.Context) {
		gotKey = defaultRedisKeyFunc(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.RemoteAddr = "198.51.100.8:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ip:198.51.100.8:GET:/api/v1/users", gotKey)
}

// TestRecovery tests the panic recovery middleware.
func TestRecovery(t *testing.T) {
	tests := []struct {
		name           string
		handler        gin.HandlerFunc
		expectedStatus int
		shouldPanic    bool
	}{
		{
			name: "recover from panic",
			handler: func(c *gin.Context) {
				panic("test panic")
			},
			expectedStatus: http.StatusInternalServerError,
			shouldPanic:    true,
		},
		{
			name: "normal request",
			handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			},
			expectedStatus: http.StatusOK,
			shouldPanic:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
			router := gin.New()
			router.Use(Recovery(RecoveryConfig{
				Logger:     log,
				StackTrace: false, // Disable stack trace for cleaner test output
			}))
			router.GET("/test", tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestGetUserID tests the GetUserID helper function.
func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*gin.Context)
		wantID string
		wantOK bool
	}{
		{
			name: "user ID exists in context",
			setup: func(c *gin.Context) {
				c.Set(UserIDKey, "user-123")
			},
			wantID: "user-123",
			wantOK: true,
		},
		{
			name:   "user ID does not exist",
			setup:  func(c *gin.Context) {},
			wantID: "",
			wantOK: false,
		},
		{
			name: "user ID is not a string",
			setup: func(c *gin.Context) {
				c.Set(UserIDKey, 123)
			},
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			tt.setup(c)

			// Act
			userID, ok := GetUserID(c)

			// Assert
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantID, userID)
		})
	}
}

// TestGetUserRole tests the GetUserRole helper function.
func TestGetUserRole(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*gin.Context)
		wantRole domain.Role
		wantOK   bool
	}{
		{
			name: "role exists in context",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, string(domain.RoleAdmin))
			},
			wantRole: domain.RoleAdmin,
			wantOK:   true,
		},
		{
			name:     "role does not exist",
			setup:    func(c *gin.Context) {},
			wantRole: "",
			wantOK:   false,
		},
		{
			name: "role is not a string",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, 123)
			},
			wantRole: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			tt.setup(c)

			// Act
			role, ok := GetUserRole(c)

			// Assert
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantRole, role)
		})
	}
}

// TestIsAuthenticated tests the IsAuthenticated helper function.
func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*gin.Context)
		want  bool
	}{
		{
			name: "user is authenticated",
			setup: func(c *gin.Context) {
				c.Set(UserIDKey, "user-123")
			},
			want: true,
		},
		{
			name:  "user is not authenticated",
			setup: func(c *gin.Context) {},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			tt.setup(c)

			// Act
			isAuth := IsAuthenticated(c)

			// Assert
			assert.Equal(t, tt.want, isAuth)
		})
	}
}

// TestIsAdmin tests the IsAdmin helper function.
func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*gin.Context)
		want  bool
	}{
		{
			name: "user is admin",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, string(domain.RoleAdmin))
			},
			want: true,
		},
		{
			name: "user is not admin",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, string(domain.RoleUser))
			},
			want: false,
		},
		{
			name:  "user is not authenticated",
			setup: func(c *gin.Context) {},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			tt.setup(c)

			// Act
			isAdmin := IsAdmin(c)

			// Assert
			assert.Equal(t, tt.want, isAdmin)
		})
	}
}

// TestCORS tests the CORS middleware.
func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		config         CORSConfig
		origin         string
		method         string
		expectedStatus int
		checkHeaders   map[string]string
	}{
		{
			name: "allow all origins with wildcard",
			config: CORSConfig{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				MaxAge:           3600,
			},
			origin:         "https://example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST",
				"Access-Control-Allow-Headers": "Content-Type",
			},
		},
		{
			name: "allow specific origin",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:         "https://example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
		{
			name: "reject disallowed origin",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:         "https://evil.com",
			method:         "GET",
			expectedStatus: http.StatusOK, // Request still succeeds, just no CORS headers
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "handle preflight request",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:         "https://example.com",
			method:         "OPTIONS",
			expectedStatus: http.StatusNoContent, // 204 for preflight
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name: "allow credentials",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowCredentials: true,
			},
			origin:         "https://example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "expose custom headers",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				ExposedHeaders: []string{"X-Custom-Header"},
				AllowedMethods: []string{"GET"},
			},
			origin:         "https://example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Expose-Headers": "X-Custom-Header",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(CORS(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			for header, expectedValue := range tt.checkHeaders {
				if expectedValue == "" {
					assert.Empty(t, w.Header().Get(header), "Header %s should be empty", header)
				} else {
					assert.Equal(t, expectedValue, w.Header().Get(header), "Header %s mismatch", header)
				}
			}
		})
	}
}

// TestCORS_DefaultConfig tests CORS with default configuration.
func TestCORS_DefaultConfig(t *testing.T) {
	// Arrange
	router := gin.New()
	router.Use(CORS(DefaultCORSConfig()))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
}

// TestRecovery_WithStackTrace tests recovery with stack trace enabled.
func TestRecovery_WithStackTrace(t *testing.T) {
	// Arrange
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	router := gin.New()
	router.Use(Recovery(RecoveryConfig{
		Logger:     log,
		StackTrace: true, // Enable stack trace
	}))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic with stack trace")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "internal_server_error", response["error"])
}

// TestRecovery_CustomErrorHandler tests recovery with custom error handler.
func TestRecovery_CustomErrorHandler(t *testing.T) {
	// Arrange
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	customCalled := false

	router := gin.New()
	router.Use(Recovery(RecoveryConfig{
		Logger: log,
		ErrorHandler: func(c *gin.Context, err interface{}) {
			customCalled = true
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "service_unavailable",
				"panic": fmt.Sprintf("%v", err),
			})
		},
	}))
	router.GET("/panic", func(c *gin.Context) {
		panic("custom handler panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.True(t, customCalled, "Custom error handler should be called")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "service_unavailable", response["error"])
	assert.Equal(t, "custom handler panic", response["panic"])
}

// TestRecovery_NoPanic tests recovery middleware without panic.
func TestRecovery_NoPanic(t *testing.T) {
	// Arrange
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	router := gin.New()
	router.Use(Recovery(RecoveryConfig{
		Logger:     log,
		StackTrace: false,
	}))
	router.GET("/normal", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/normal", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["message"])
}

// TestSlidingWindowRateLimit tests sliding window rate limiter.
func TestSlidingWindowRateLimit(t *testing.T) {
	// Arrange
	router := gin.New()
	router.Use(SlidingWindowRateLimit(2, 10*time.Second)) // 2 requests per 10 seconds
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act - Make 3 requests rapidly
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	req3 := httptest.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	// Assert
	assert.Equal(t, http.StatusOK, w1.Code, "First request should succeed")
	assert.Equal(t, http.StatusOK, w2.Code, "Second request should succeed")
	assert.Equal(t, http.StatusTooManyRequests, w3.Code, "Third request should be rate limited")
}

// TestRateLimit_ConcurrentRequests tests rate limiting with concurrent requests.
func TestRateLimit_ConcurrentRequests(t *testing.T) {
	// Arrange
	router := gin.New()
	router.Use(RateLimit(RateLimiterConfig{
		RequestsPerSecond: 1, // 1 request per second
		Burst:             1, // Allow 1 burst
	}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act - Make concurrent requests
	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	// Collect results
	var successCount, rateLimitedCount int
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusOK {
			successCount++
		} else if status == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Assert - Some requests should succeed, some should be rate limited
	assert.Greater(t, successCount, 0, "At least one request should succeed")
	assert.Greater(t, rateLimitedCount, 0, "Some requests should be rate limited")
	assert.Equal(t, numRequests, successCount+rateLimitedCount, "All requests should be accounted for")
}

// TestRateLimit_KeyFunc tests custom key function for rate limiting.
func TestRateLimit_KeyFunc(t *testing.T) {
	// Arrange
	router := gin.New()
	router.Use(RateLimit(RateLimiterConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		KeyFunc: func(c *gin.Context) string {
			// Rate limit by API key instead of IP
			apiKey := c.GetHeader("X-API-Key")
			if apiKey == "" {
				apiKey = "default"
			}
			return apiKey
		},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act - Make requests with different API keys
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-API-Key", "key1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-API-Key", "key2")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.Header.Set("X-API-Key", "key1")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	// Assert - Both keys should work, second request with key1 should be rate limited
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
}

// TestRateLimit_Cleanup tests rate limiter cleanup functionality.
func TestRateLimit_Cleanup(t *testing.T) {
	// Arrange
	config := RateLimiterConfig{
		RequestsPerSecond: 1,
		Burst:             1,
	}
	limiter := NewIPRateLimiter(config)

	// Act - Add an entry
	key := "test-ip"
	l := limiter.getLimiter(key)
	assert.NotNil(t, l)

	// Wait a moment and trigger cleanup
	time.Sleep(100 * time.Millisecond)
	limiter.cleanupOldLimiters(1 * time.Millisecond)

	// Assert - Entry should be cleaned up
	limiter.mu.RLock()
	_, exists := limiter.limiters[key]
	limiter.mu.RUnlock()
	assert.False(t, exists, "Old limiter entry should be cleaned up")
}

// TestRateLimit_SkipFunc tests rate limiter skip function.
func TestRateLimit_SkipFunc(t *testing.T) {
	tests := []struct {
		name     string
		skipFunc func(*gin.Context) bool
		headers  map[string]string
		skipped  bool
	}{
		{
			name: "skip by header",
			skipFunc: func(c *gin.Context) bool {
				return c.GetHeader("X-Bypass-RateLimit") == "true"
			},
			headers: map[string]string{
				"X-Bypass-RateLimit": "true",
			},
			skipped: true,
		},
		{
			name: "do not skip",
			skipFunc: func(c *gin.Context) bool {
				return c.GetHeader("X-Bypass-RateLimit") == "true"
			},
			headers: map[string]string{},
			skipped: false,
		},
		{
			name: "skip by path",
			skipFunc: func(c *gin.Context) bool {
				return c.Request.URL.Path == "/health"
			},
			headers: map[string]string{},
			skipped: false, // Request goes to /test, not /health
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			router.Use(RateLimit(RateLimiterConfig{
				RequestsPerSecond: 0.1, // Very low limit
				Burst:             1,
				SkipFunc:          tt.skipFunc,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Make first request to consume burst
			req1 := httptest.NewRequest("GET", "/test", nil)
			w1 := httptest.NewRecorder()
			router.ServeHTTP(w1, req1)

			// Make second request with headers
			req2 := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req2.Header.Set(k, v)
			}
			w2 := httptest.NewRecorder()
			router.ServeHTTP(w2, req2)

			// Assert
			if tt.skipped {
				assert.Equal(t, http.StatusOK, w2.Code, "Request should be skipped from rate limiting")
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w2.Code, "Request should be rate limited")
			}
		})
	}
}

// TestMiddlewareRegistry tests the middleware registry.
func TestMiddlewareRegistry(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Env: "development"},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{Secret: "test-secret"},
		},
	}
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	registry := NewMiddlewareRegistry(cfg, log)
	assert.NotNil(t, registry)
}

// TestDefaultMiddleware tests default middleware chain.
func TestDefaultMiddleware(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Env: "development"},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{Secret: "test-secret"},
		},
	}
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	registry := NewMiddlewareRegistry(cfg, log)
	middleware := registry.DefaultMiddleware()
	assert.NotNil(t, middleware)
	assert.Len(t, middleware, 4)
}

// TestRegistryMiddlewareMethods tests all registry middleware methods.
func TestRegistryMiddlewareMethods(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Env: "production"},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{Secret: "test-secret"},
		},
	}
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	registry := NewMiddlewareRegistry(cfg, log)

	assert.NotNil(t, registry.RequestID())
	assert.NotNil(t, registry.Logger())
	assert.NotNil(t, registry.Recovery())
	assert.NotNil(t, registry.CORS())
	assert.NotNil(t, registry.Auth("/health"))
	assert.NotNil(t, registry.RateLimit(10.0, 20))
	assert.NotNil(t, registry.Timeout(30*time.Second))
	assert.NotNil(t, registry.AdminOnly())

	routes := registry.PublicRoutes()
	assert.Contains(t, routes, "/health")

	healthRoutes := registry.HealthCheckRoutes()
	assert.Contains(t, healthRoutes, "/health")
}

// TestCustomRecovery tests custom recovery middleware.
func TestCustomRecovery(t *testing.T) {
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	customCalled := false
	customHandler := func(c *gin.Context, err interface{}) {
		customCalled = true
		c.JSON(500, gin.H{"custom": "handled"})
	}

	router := gin.New()
	router.Use(CustomRecovery(log, customHandler))
	router.GET("/panic", func(c *gin.Context) {
		panic("test")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, customCalled)
}

// TestRecoveryWithWriter tests recovery with writer.
func TestRecoveryWithWriter(t *testing.T) {
	log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
	router := gin.New()
	router.Use(RecoveryWithWriter(log))
	router.GET("/panic", func(c *gin.Context) {
		panic("test")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestPanicHandler tests panic handler function.
func TestPanicHandler(t *testing.T) {
	tests := []struct {
		name          string
		detail        bool
		hasPanicField bool
	}{
		{"with detail", true, true},
		{"without detail", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := logger.New(&logger.Config{Level: "error", Format: "console"})
			handler := PanicHandler(tt.detail)
			router := gin.New()
			router.Use(Recovery(RecoveryConfig{
				Logger:       log,
				ErrorHandler: handler,
			}))
			router.GET("/panic", func(c *gin.Context) {
				panic("error")
			})

			req := httptest.NewRequest("GET", "/panic", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			_, hasPanic := response["panic"]
			assert.Equal(t, tt.hasPanicField, hasPanic)
		})
	}
}

// TestTimeoutMiddleware tests timeout middleware.
// Note: These tests are timing-dependent and may be flaky in CI environments.
// Skip if running in short test mode.
func TestTimeoutMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing-sensitive test in short mode")
	}

	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectedStatus int
	}{
		// Use more lenient timeouts to avoid flakiness
		{"completes in time", 200 * time.Millisecond, 50 * time.Millisecond, http.StatusOK},
		{"times out", 50 * time.Millisecond, 200 * time.Millisecond, http.StatusRequestTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Timeout(TimeoutConfig{Timeout: tt.timeout}))
			router.GET("/test", func(c *gin.Context) {
				time.Sleep(tt.handlerDelay)
				c.JSON(200, gin.H{"ok": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestTimeoutWithHandler tests timeout with custom handler.
func TestTimeoutWithHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing-sensitive test in short mode")
	}

	customCalled := false
	router := gin.New()
	router.Use(TimeoutWithHandler(50*time.Millisecond, func(c *gin.Context) {
		customCalled = true
		c.JSON(503, gin.H{"timeout": true})
	}))
	router.GET("/test", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, customCalled)
}

// TestRequestTimeout tests request timeout helper.
func TestRequestTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing-sensitive test in short mode")
	}

	router := gin.New()
	router.Use(RequestTimeout(5 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.JSON(200, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
}

// TestLoggingMiddleware tests logging middleware.
func TestLoggingMiddleware(t *testing.T) {
	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	router := gin.New()
	router.Use(Logging(LoggingConfig{
		Logger:    log,
		SkipPaths: []string{"/health"},
	}))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest("GET", "/health", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestErrorLoggingMiddleware tests error logging middleware.
func TestErrorLoggingMiddleware(t *testing.T) {
	log, _ := logger.New(&logger.Config{Level: "error", Format: "json"})
	router := gin.New()
	router.Use(ErrorLogging(log))
	router.GET("/error", func(c *gin.Context) { c.JSON(500, gin.H{}) })
	router.GET("/ok", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	req1 := httptest.NewRequest("GET", "/error", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusInternalServerError, w1.Code)

	req2 := httptest.NewRequest("GET", "/ok", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}
