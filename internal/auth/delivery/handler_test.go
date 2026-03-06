// Package delivery tests HTTP handlers for the auth service.
package delivery_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	authusecasemocks "github.com/ignata/go-microservices-boilerplate/internal/auth/usecase/mocks"
)

// setupTestRouter creates a test router with Gin in test mode.
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// TestRegister_Success tests successful user registration.
func TestRegister_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Role:     "USER",
			IsActive: true,
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.Equal(t, "test@example.com", data["user"].(map[string]interface{})["email"])

	mockUseCase.AssertExpectations(t)
}

// TestRegister_ValidationError tests registration with invalid input.
func TestRegister_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"password": "SecurePass123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":    "invalid-email",
				"password": "SecurePass123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			var bodyBytes []byte
			if tt.requestBody == nil {
				bodyBytes = []byte("{invalid json")
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			// Act
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.POST("/auth/register", handler.Register)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.False(t, response["success"].(bool))
		})
	}
}

// TestRegister_EmailAlreadyUsed tests registration with existing email.
func TestRegister_EmailAlreadyUsed(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, domain.ErrEmailAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"email":    "existing@example.com",
		"password": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["error"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "already in use")

	mockUseCase.AssertExpectations(t)
}

// TestRegister_WithRole tests registration with role specified.
func TestRegister_WithRole(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "admin@example.com",
			Role:     "ADMIN",
			IsActive: true,
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.MatchedBy(func(r *dto.RegisterRequest) bool {
		return r.Email == "admin@example.com" && r.Role == "ADMIN"
	})).Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":    "admin@example.com",
		"password": "SecurePass123",
		"role":     "ADMIN",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	assert.Equal(t, "ADMIN", user["role"])

	mockUseCase.AssertExpectations(t)
}

// TestLogin_Success tests successful user login.
func TestLogin_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Role:     "USER",
			IsActive: true,
		},
	}

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "CorrectPassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])

	mockUseCase.AssertExpectations(t)
}

// TestLogin_InvalidCredentials tests login with invalid credentials.
func TestLogin_InvalidCredentials(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil, domain.ErrInvalidCredentials)

	// Act
	reqBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "WrongPassword",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestLogin_UserDeleted tests login with deleted user.
func TestLogin_UserDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil, domain.ErrUserDeleted)

	// Act
	reqBody := map[string]interface{}{
		"email":    "deleted@example.com",
		"password": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestLogin_UserInactive tests login with inactive user.
func TestLogin_UserInactive(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil, domain.ErrUserInactive)

	// Act
	reqBody := map[string]interface{}{
		"email":    "inactive@example.com",
		"password": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestLogout_Success tests successful logout.
func TestLogout_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Logout", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").Return(nil)

	// Act
	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	// Set user_id in context (simulating auth middleware)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/logout", handler.Logout)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Successfully logged out", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestLogout_Unauthorized tests logout without authentication.
func TestLogout_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - no user_id set in context
	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.POST("/auth/logout", handler.Logout)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestRefreshToken_Success tests successful token refresh.
func TestRefreshToken_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Role:     "USER",
			IsActive: true,
		},
	}

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"refresh_token": "valid-refresh-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "new-access-token", data["access_token"])

	mockUseCase.AssertExpectations(t)
}

// TestRefreshToken_InvalidToken tests refresh with invalid token.
func TestRefreshToken_InvalidToken(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(nil, domain.ErrInvalidToken)

	// Act
	reqBody := map[string]interface{}{
		"refresh_token": "invalid-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetCurrentUser_Success tests getting current user.
func TestGetCurrentUser_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UserResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "test@example.com",
		Role:     "USER",
		IsActive: true,
	}

	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	// Set user_id in context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)
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

// TestGetCurrentUser_Unauthorized tests getting current user without auth.
func TestGetCurrentUser_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - no user_id set
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	router.GET("/auth/me", handler.GetCurrentUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestChangePassword_Success tests successful password change.
func TestChangePassword_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.ChangePasswordRequest")).
		Return(nil)

	// Act
	reqBody := map[string]interface{}{
		"current_password": "OldPass123",
		"new_password":     "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user_id in context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Password changed successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestChangePassword_InvalidCurrentPassword tests password change with wrong current password.
func TestChangePassword_InvalidCurrentPassword(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.ChangePasswordRequest")).
		Return(domain.ErrInvalidPassword)

	// Act
	reqBody := map[string]interface{}{
		"current_password": "WrongPass",
		"new_password":     "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user_id in context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestChangePassword_Unauthorized tests password change without auth.
func TestChangePassword_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - no user_id set
	reqBody := map[string]interface{}{
		"current_password": "OldPass123",
		"new_password":     "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/change-password", handler.ChangePassword)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGetUser_Success tests getting user by ID (admin).
func TestGetUser_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UserResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "test@example.com",
		Role:     "USER",
		IsActive: true,
	}

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
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

// TestGetUser_NotFound tests getting non-existent user.
func TestGetUser_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(nil, domain.ErrUserNotFound)

	// Act
	req, _ := http.NewRequest("GET", "/admin/users/550e8400-e29b-41d4-a716-446655440002", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListUsers_Success tests listing users.
func TestListUsers_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UserListResponse{
		Users: []*dto.UserResponse{
			{ID: "user-1", Email: "user1@example.com", Role: "USER", IsActive: true},
			{ID: "user-2", Email: "user2@example.com", Role: "ADMIN", IsActive: true},
		},
		Pagination: &dto.PaginationMeta{
			Page:       1,
			Limit:      10,
			Total:      2,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/admin/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["users"])

	mockUseCase.AssertExpectations(t)
}

// TestListUsers_WithFilters tests listing users with filters.
func TestListUsers_WithFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		setup func(*authusecasemocks.AuthUseCase)
	}{
		{
			name:  "filter by role",
			query: "/admin/users?role=ADMIN",
			setup: func(m *authusecasemocks.AuthUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.Role == "ADMIN"
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "filter by search",
			query: "/admin/users?search=test@example.com",
			setup: func(m *authusecasemocks.AuthUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.Search == "test@example.com"
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "include deleted",
			query: "/admin/users?include_deleted=true",
			setup: func(m *authusecasemocks.AuthUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.IncludeDeleted == true
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
		{
			name:  "only deleted",
			query: "/admin/users?only_deleted=true",
			setup: func(m *authusecasemocks.AuthUseCase) {
				m.On("ListUsers", mock.Anything, mock.MatchedBy(func(r *dto.ListUsersRequest) bool {
					return r.OnlyDeleted == true
				})).Return(&dto.UserListResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			tt.setup(mockUseCase)

			// Act
			req, _ := http.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/admin/users", handler.ListUsers)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

// TestDeleteUser_Success tests successful user deletion (soft delete).
func TestDeleteUser_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.MatchedBy(func(r *dto.DeleteUserRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Force == false
	})).Return(nil)

	// Act
	req, _ := http.NewRequest("DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
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

// TestDeleteUser_ForceDelete tests forced user deletion (hard delete).
func TestDeleteUser_ForceDelete(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.MatchedBy(func(r *dto.DeleteUserRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Force == true
	})).Return(nil)

	// Act
	req, _ := http.NewRequest("DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001?force=true", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User permanently deleted", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestDeleteUser_NotFound tests deleting non-existent user.
func TestDeleteUser_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(domain.ErrUserNotFound)

	// Act
	req, _ := http.NewRequest("DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440002", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreUser_Success tests successful user restoration.
func TestRestoreUser_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedUser := &dto.UserResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "test@example.com",
		Role:     "USER",
		IsActive: true,
	}

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(expectedUser, nil)

	// Act
	req, _ := http.NewRequest("POST", "/admin/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/admin/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "User restored successfully", data["message"])
	assert.NotNil(t, data["user"])

	mockUseCase.AssertExpectations(t)
}

// TestRestoreUser_NotFound tests restoring non-existent user.
func TestRestoreUser_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(nil, domain.ErrUserNotFound)

	// Act
	req, _ := http.NewRequest("POST", "/admin/users/550e8400-e29b-41d4-a716-446655440002/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/admin/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_ValidationError tests validation error handling.
func TestHandleError_ValidationError(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// No mock expectation needed - validation fails at handler level before usecase is called

	// Act
	reqBody := map[string]interface{}{
		"email":    "invalid-email",
		"password": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestHandleError_SessionExpired tests session expired error handling.
func TestHandleError_SessionExpired(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(nil, domain.ErrSessionExpired)

	// Act
	reqBody := map[string]interface{}{
		"refresh_token": "expired-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_UserGone tests handling deleted user error (410 Gone).
func TestHandleError_UserGone(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(nil, domain.ErrUserDeleted)

	// Act
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	// Set user_id in context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusGone, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}
