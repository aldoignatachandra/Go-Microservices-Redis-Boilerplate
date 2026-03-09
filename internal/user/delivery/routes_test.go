package delivery_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery/mocks"
)

// TestCORSMiddleware tests the CORS middleware.
func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter(nil)
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
	delivery.RegisterRoutes(router, handler)

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
