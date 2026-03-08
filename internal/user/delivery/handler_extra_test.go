package delivery_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/profile", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id/profile", handler.GetProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("GetProfile", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, domain.ErrProfileNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/profile", nil)
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
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
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

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/activate", nil)
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

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/deactivate", nil)
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

	req, _ := http.NewRequest("DELETE", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
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

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
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

// TestUpdateProfile_Unauthorized tests profile update without authentication.
func TestUpdateProfile_Unauthorized(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	reqBody := map[string]interface{}{"first_name": "John"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/v1/users/profile", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/api/v1/users/profile", handler.UpdateProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGetUser_IncludeDeleted tests getting a deleted user.
func TestGetUser_IncludeDeleted(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	deletedTime, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	expectedUser := &dto.UserResponse{
		ID:        "550e8400-e29b-41d4-a716-446655440001",
		Email:     "deleted@example.com",
		Role:      "USER",
		IsActive:  false,
		DeletedAt: &deletedTime,
	}

	mockUseCase.On("GetUser", mock.Anything, mock.MatchedBy(func(r *dto.GetUserRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.IncludeDeleted == true
	})).Return(expectedUser, nil)

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001?include_deleted=true", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListUsers_WithFilters tests listing users with filters.
func TestListUsers_WithFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		setup func(*mocks.MockUserUseCase)
	}{
		{
			name:  "filter by role",
			query: "/api/v1/users?role=ADMIN",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.Role == "ADMIN"
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "filter by search",
			query: "/api/v1/users?search=test@example.com",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.Search == "test@example.com"
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "include deleted",
			query: "/api/v1/users?include_deleted=true",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.IncludeDeleted == true
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "only deleted",
			query: "/api/v1/users?only_deleted=true",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.OnlyDeleted == true
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "pagination",
			query: "/api/v1/users?page=2&limit=50",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.Page == 2 && r.Limit == 50
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := new(mocks.MockUserUseCase)
			handler := delivery.NewUserHandler(mockUseCase)
			router := setupTestRouter(handler)

			tt.setup(mockUseCase)

			req, _ := http.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/api/v1/users", handler.ListUsers)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

// TestDeleteUser_ForceDelete tests forced user deletion.
func TestDeleteUser_ForceDelete(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("DeleteUser", mock.Anything, mock.MatchedBy(func(r *dto.DeleteUserRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Force == true
	})).Return(nil)

	req, _ := http.NewRequest("DELETE", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001?force=true", nil)
	w := httptest.NewRecorder()

	router.DELETE("/api/v1/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreUser_AlreadyActive tests restoring already active user.
func TestRestoreUser_AlreadyActive(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	expectedResponse := &dto.RestoreResponse{
		Success: false,
		Message: "User is already active",
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Role:     "USER",
			IsActive: true,
		},
	}

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(expectedResponse, nil)

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "User is already active", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestGetActivityLogs_WithFilters tests activity logs with filters.
func TestGetActivityLogs_WithFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		setup func(*mocks.MockUserUseCase)
	}{
		{
			name:  "filter by user_id",
			query: "/api/v1/activity-logs?user_id=550e8400-e29b-41d4-a716-446655440001",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("GetActivityLogs", mock.Anything, mock.MatchedBy(func(r *dto.ListActivityLogsRequest) bool {
					return r.UserID == "550e8400-e29b-41d4-a716-446655440001"
				})).Return(&dto.ActivityLogListResponse{}, nil)
			},
		},
		{
			name:  "filter by action",
			query: "/api/v1/activity-logs?action=login",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("GetActivityLogs", mock.Anything, mock.MatchedBy(func(r *dto.ListActivityLogsRequest) bool {
					return r.Action == "login"
				})).Return(&dto.ActivityLogListResponse{}, nil)
			},
		},
		{
			name:  "filter by resource",
			query: "/api/v1/activity-logs?resource=auth",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("GetActivityLogs", mock.Anything, mock.MatchedBy(func(r *dto.ListActivityLogsRequest) bool {
					return r.Resource == "auth"
				})).Return(&dto.ActivityLogListResponse{}, nil)
			},
		},
		{
			name:  "pagination",
			query: "/api/v1/activity-logs?page=2&limit=50",
			setup: func(m *mocks.MockUserUseCase) {
				m.On("GetActivityLogs", mock.Anything, mock.MatchedBy(func(r *dto.ListActivityLogsRequest) bool {
					return r.Page == 2 && r.Limit == 50
				})).Return(&dto.ActivityLogListResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := new(mocks.MockUserUseCase)
			handler := delivery.NewUserHandler(mockUseCase)
			router := setupTestRouter(handler)

			tt.setup(mockUseCase)

			req, _ := http.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/api/v1/activity-logs", handler.GetActivityLogs)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

// TestUpdateProfile_WithRequestInfo tests profile update with IP and UserAgent.
func TestUpdateProfile_WithRequestInfo(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	var capturedReq *dto.UpdateProfileRequest
	mockUseCase.On("UpdateProfile", mock.Anything, mock.MatchedBy(func(r *dto.UpdateProfileRequest) bool {
		capturedReq = r
		return r.IPAddress == "127.0.0.1" && r.UserAgent == "test-agent"
	})).Return(nil)

	reqBody := map[string]interface{}{"first_name": "John"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/v1/users/profile", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.PUT("/api/v1/users/profile", handler.UpdateProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedReq)
	assert.Equal(t, "127.0.0.1", capturedReq.IPAddress)
	assert.Equal(t, "test-agent", capturedReq.UserAgent)

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProfile_InternalError tests internal server error scenario.
func TestUpdateProfile_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*dto.UpdateProfileRequest")).
		Return(errors.New("database connection failed"))

	reqBody := map[string]interface{}{"first_name": "John"}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/v1/users/profile", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.PUT("/api/v1/users/profile", handler.UpdateProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestGetProfile_InternalError tests internal server error on get profile.
func TestGetProfile_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("GetProfile", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/profile", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id/profile", handler.GetProfile)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestGetUser_InternalError tests internal server error on get user.
func TestGetUser_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.GET("/api/v1/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestActivateUser_InternalError tests internal server error on activate.
func TestActivateUser_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("ActivateUser", mock.Anything, mock.AnythingOfType("*dto.ActivateUserRequest")).
		Return(errors.New("database error"))

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/activate", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/activate", handler.ActivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestDeactivateUser_InternalError tests internal server error on deactivate.
func TestDeactivateUser_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("DeactivateUser", mock.Anything, mock.AnythingOfType("*dto.DeactivateUserRequest")).
		Return(errors.New("database error"))

	req, _ := http.NewRequest("POST", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001/deactivate", nil)
	w := httptest.NewRecorder()

	router.POST("/api/v1/users/:id/deactivate", handler.DeactivateUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestDeleteUser_InternalError tests internal server error on delete.
func TestDeleteUser_InternalError(t *testing.T) {
	mockUseCase := new(mocks.MockUserUseCase)
	handler := delivery.NewUserHandler(mockUseCase)
	router := setupTestRouter(handler)

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(errors.New("database error"))

	req, _ := http.NewRequest("DELETE", "/api/v1/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/api/v1/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

// Helper function to create a string pointer.
func ptrString(s string) *string {
	return &s
}
