package delivery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery/mocks"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// TestCORSMiddleware tests the CORS middleware.
func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(delivery.CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	// Test OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Test GET request
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "*", w2.Header().Get("Access-Control-Allow-Origin"))
}

// TestRegisterRoutes tests route registration.
func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	delivery.RegisterRoutes(router, handler, "test-secret-key-at-least-32-chars-long!!", nil)

	routes := router.Routes()
	assert.NotNil(t, routes)

	// Check that some expected routes exist
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	// Check for user-specific routes (note: routes are under /api/v1/users/...)
	assert.True(t, routePaths["/api/v1/users/:id"], "Get user route should be registered")
	assert.True(t, routePaths["/api/v1/users"], "List users route should be registered")

	// Verify we have routes registered
	assert.Greater(t, len(routes), 0, "Should have routes registered")
}

func TestRegisterRoutes_ListUsersForbiddenForUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	const jwtSecret = "test-secret-key-at-least-32-chars-long!!"
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	delivery.RegisterRoutes(router, handler, jwtSecret, nil)

	token := mustGenerateToken(t, jwtSecret, "550e8400-e29b-41d4-a716-446655440099", "USER")
	req, _ := http.NewRequest("GET", "/api/v1/users?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "ListUsers", mock.Anything, mock.Anything)
}

func TestRegisterRoutes_ActivityLogsForbiddenForUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	const jwtSecret = "test-secret-key-at-least-32-chars-long!!"
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	delivery.RegisterRoutes(router, handler, jwtSecret, nil)

	token := mustGenerateToken(t, jwtSecret, "550e8400-e29b-41d4-a716-446655440099", "USER")
	req, _ := http.NewRequest("GET", "/api/v1/activity-logs?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "GetActivityLogs", mock.Anything, mock.Anything)
}

func TestRegisterRoutes_GetUserAllowedForSelfUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	const jwtSecret = "test-secret-key-at-least-32-chars-long!!"
	userID := "550e8400-e29b-41d4-a716-446655440099"
	mockUseCase := new(mocks.MockUserUseCase)
	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(&dto.UserResponse{ID: userID, Email: "user@example.com", Role: "USER"}, nil)

	handler := delivery.NewUserHandler(mockUseCase)
	delivery.RegisterRoutes(router, handler, jwtSecret, nil)

	token := mustGenerateToken(t, jwtSecret, userID, "USER")
	req, _ := http.NewRequest("GET", "/api/v1/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

func mustGenerateToken(t *testing.T, secret, userID, role string) string {
	t.Helper()

	manager := utils.NewJWTManager(utils.JWTConfig{
		Secret:           secret,
		ExpiresIn:        time.Hour,
		RefreshExpiresIn: time.Hour * 24,
	})

	token, err := manager.GenerateToken(userID, "test@example.com", role)
	require.NoError(t, err)
	return token
}
