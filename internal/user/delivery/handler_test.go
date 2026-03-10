// Package delivery tests HTTP handlers for the user service.
package delivery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	mocks "github.com/ignata/go-microservices-boilerplate/internal/user/delivery/mocks"
	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func withAuthContext(userID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_role", role)
		c.Next()
	}
}

// TestGetUser_Success tests successful user retrieval.
func TestGetUser_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	expectedUser := &dto.UserResponse{
		ID:    "550e8400-e29b-41d4-a716-446655440001",
		Email: "test@example.com",
		Role:  "USER",
	}

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(expectedUser, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.GET("/api/v1/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])

	mockUseCase.AssertExpectations(t)
}

// TestGetUser_NotFound tests user not found scenario.
func TestGetUser_NotFound(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, domain.ErrUserNotFound)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/users/non-existent", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.GET("/api/v1/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "not found")

	mockUseCase.AssertExpectations(t)
}

// TestGetUser_ForbiddenForNonOwner ensures USER cannot access another user's profile.
func TestGetUser_ForbiddenForNonOwner(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440099")
		c.Set("user_role", "USER")
		c.Next()
	})

	router.GET("/api/v1/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "GetUser", mock.Anything, mock.Anything)
}

// TestListUsers_Success tests successful user list retrieval.
func TestListUsers_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	expectedUsers := &dto.UserListResponse{
		Data: []*dto.UserResponse{
			{ID: "user-1", Email: "user1@example.com", Role: "USER"},
			{ID: "user-2", Email: "user2@example.com", Role: "ADMIN"},
		},
		Pagination: &dto.PaginationMeta{
			Page:            1,
			Limit:           20,
			Total:           2,
			TotalPages:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(expectedUsers, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/users?page=1&limit=20", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.GET("/api/v1/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["data"])

	mockUseCase.AssertExpectations(t)
}

// TestActivateUser_Success tests successful user activation.
func TestActivateUser_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ActivateUser", mock.Anything, mock.AnythingOfType("*dto.ActivateUserRequest")).
		Return(nil)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/activate", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.POST("/api/v1/users/:id/activate", handler.ActivateUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User activated successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

func TestActivateUser_ForbiddenForSelf(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	targetID := "550e8400-e29b-41d4-a716-446655440001"
	req, _ := http.NewRequest("POST", "/api/v1/users/"+targetID+"/activate", nil)
	w := httptest.NewRecorder()

	router.Use(withAuthContext(targetID, "ADMIN"))
	router.POST("/api/v1/users/:id/activate", handler.ActivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "ActivateUser", mock.Anything, mock.Anything)
}

// TestDeactivateUser_Success tests successful user deactivation.
func TestDeactivateUser_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeactivateUser", mock.Anything, mock.AnythingOfType("*dto.DeactivateUserRequest")).
		Return(nil)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/deactivate", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.POST("/api/v1/users/:id/deactivate", handler.DeactivateUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User deactivated successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

func TestDeactivateUser_ForbiddenForSelf(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	targetID := "550e8400-e29b-41d4-a716-446655440001"
	req, _ := http.NewRequest("POST", "/api/v1/users/"+targetID+"/deactivate", nil)
	w := httptest.NewRecorder()

	router.Use(withAuthContext(targetID, "ADMIN"))
	router.POST("/api/v1/users/:id/deactivate", handler.DeactivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "DeactivateUser", mock.Anything, mock.Anything)
}

// TestDeleteUser_Success tests successful user deletion.
func TestDeleteUser_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(nil)

	// Create request
	req, _ := http.NewRequest("DELETE", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.DELETE("/api/v1/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User deleted successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

func TestDeleteUser_ForbiddenForSelf(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	targetID := "550e8400-e29b-41d4-a716-446655440001"
	req, _ := http.NewRequest("DELETE", "/api/v1/users/"+targetID, nil)
	w := httptest.NewRecorder()

	router.Use(withAuthContext(targetID, "ADMIN"))
	router.DELETE("/api/v1/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.Anything)
}

// TestRestoreUser_Success tests successful user restoration.
func TestRestoreUser_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.RestoreResponse{
		Success: true,
		Message: "User restored successfully",
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(expectedResponse, nil)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.POST("/api/v1/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User restored successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

func TestRestoreUser_ForbiddenForSelf(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	targetID := "550e8400-e29b-41d4-a716-446655440001"
	req, _ := http.NewRequest("POST", "/api/v1/users/"+targetID+"/restore", nil)
	w := httptest.NewRecorder()

	router.Use(withAuthContext(targetID, "ADMIN"))
	router.POST("/api/v1/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockUseCase.AssertNotCalled(t, "RestoreUser", mock.Anything, mock.Anything)
}

// TestGetActivityLogs_Success tests successful activity logs retrieval.
func TestGetActivityLogs_Success(t *testing.T) {
	// Setup mock
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter()

	expectedLogs := &dto.ActivityLogListResponse{
		Data: []*dto.ActivityLogResponse{
			{
				ID:     "log-1",
				UserID: "550e8400-e29b-41d4-a716-446655440001",
				Action: "login",
				Entity: "auth",
			},
		},
		Pagination: &dto.PaginationMeta{
			Page:       1,
			Limit:      20,
			Total:      1,
			TotalPages: 1,
		},
	}

	mockUseCase.On("GetActivityLogs", mock.Anything, mock.AnythingOfType("*dto.ListActivityLogsRequest")).
		Return(expectedLogs, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/activity-logs?page=1&limit=20", nil)
	w := httptest.NewRecorder()

	// Register route and serve
	router.GET("/api/v1/activity-logs", handler.GetActivityLogs)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["data"])

	mockUseCase.AssertExpectations(t)
}
