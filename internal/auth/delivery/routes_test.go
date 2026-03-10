// Package delivery tests route registration for the auth service.
package delivery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/delivery"
	authusecasemocks "github.com/ignata/go-microservices-boilerplate/internal/auth/usecase/mocks"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// TestPublicHealth tests public health endpoint.
func TestPublicHealth(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/health", handler.PublicHealth)

	// Act
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestReadyProbe tests readiness probe endpoint.
func TestReadyProbe(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/ready", handler.ReadyProbe)

	// Act
	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestLiveProbe tests liveness probe endpoint.
func TestLiveProbe(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/live", handler.LiveProbe)

	// Act
	req, _ := http.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRegisterRoutes tests route registration.
func TestRegisterRoutes(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockUseCase := new(authusecasemocks.AuthUseCase)
	jwtSecret := "test-secret"

	// Act - This should not panic
	delivery.RegisterRoutes(router, mockUseCase, jwtSecret, nil)

	// Assert - Verify routes are registered
	routes := router.Routes()
	assert.NotNil(t, routes)

	// Check that some expected routes exist
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	assert.False(t, routePaths["/health"], "Health route should not be registered by delivery routes")
	assert.True(t, routePaths["/api/v1/auth/register"], "Register route should be registered")
	assert.True(t, routePaths["/api/v1/auth/login"], "Login route should be registered")
	assert.True(t, routePaths["/api/v1/users"], "Admin user list route should be registered")
	assert.False(t, routePaths["/admin/users"], "Legacy admin users route should not be registered")
}

// TestPublicHealth_Response tests public health response format.
func TestPublicHealth_Response(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/health", handler.PublicHealth)

	// Act
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "service-auth", response["service"])
}

// TestReadyProbe_Response tests readiness probe response format.
func TestReadyProbe_Response(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/ready", handler.ReadyProbe)

	// Act
	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["ready"])
}

// TestLiveProbe_Response tests liveness probe response format.
func TestLiveProbe_Response(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/live", handler.LiveProbe)

	// Act
	req, _ := http.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["alive"])
}

func TestRegisterRoute_RequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockUseCase := new(authusecasemocks.AuthUseCase)
	delivery.RegisterRoutes(router, mockUseCase, "test-secret", nil)

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRegisterRoute_RejectsUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	const jwtSecret = "test-secret-key-at-least-32-chars-long!!"
	mockUseCase := new(authusecasemocks.AuthUseCase)
	delivery.RegisterRoutes(router, mockUseCase, jwtSecret, nil)

	token := mustGenerateAuthToken(t, jwtSecret, "550e8400-e29b-41d4-a716-446655440010", "USER")
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func mustGenerateAuthToken(t *testing.T, secret, userID, role string) string {
	t.Helper()

	manager := utils.NewJWTManager(utils.JWTConfig{
		Secret:           secret,
		ExpiresIn:        time.Hour,
		RefreshExpiresIn: 24 * time.Hour,
	})

	token, err := manager.GenerateToken(userID, "test@example.com", role)
	require.NoError(t, err)
	return token
}
