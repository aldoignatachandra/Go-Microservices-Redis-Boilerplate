package delivery_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/user/delivery/mocks"
	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
)

func TestGetProfile_Success(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	expectedProfile := &dto.ProfileResponse{
		ID:        "prof-123",
		FirstName: "John",
		LastName:  "Doe",
		FullName:  "John Doe",
	}

	mockUseCase.On("GetProfile", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(expectedProfile, nil)

	req, _ := http.NewRequest("GET", "/api/v1/users/user-123/profile", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id/profile", handler.GetProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("GetProfile", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, domain.ErrProfileNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/users/user-123/profile", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id/profile", handler.GetProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*dto.UpdateProfileRequest")).
		Return(domain.ErrProfileNotFound)

	reqBody := map[string]interface{}{"first_name": "John"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/v1/users/profile", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})

	router.PUT("/api/v1/users/profile", handler.UpdateProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestActivateUser_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("ActivateUser", mock.Anything, mock.AnythingOfType("*dto.ActivateUserRequest")).
		Return(domain.ErrUserNotFound)

	req, _ := http.NewRequest("POST", "/api/v1/users/user-123/activate", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/activate", handler.ActivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestDeactivateUser_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("DeactivateUser", mock.Anything, mock.AnythingOfType("*dto.DeactivateUserRequest")).
		Return(domain.ErrUserNotFound)

	req, _ := http.NewRequest("POST", "/api/v1/users/user-123/deactivate", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/deactivate", handler.DeactivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestDeleteUser_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(domain.ErrUserNotFound)

	req, _ := http.NewRequest("DELETE", "/api/v1/users/user-123", nil)
	w := httptest.NewRecorder()

	router.DELETE("/api/v1/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestRestoreUser_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(nil, errors.New("internal server error"))

	req, _ := http.NewRequest("POST", "/api/v1/users/user-123/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestGetActivityLogs_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("GetActivityLogs", mock.Anything, mock.AnythingOfType("*dto.ListActivityLogsRequest")).
		Return(nil, errors.New("internal server error"))

	req, _ := http.NewRequest("GET", "/api/v1/activity-logs", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/activity-logs", handler.GetActivityLogs)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestListUsers_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(nil, errors.New("internal error"))

	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}
