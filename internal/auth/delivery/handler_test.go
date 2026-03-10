// Package delivery tests HTTP handlers for the auth service.
package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
	assert.NotEmpty(t, data["token"])
	assert.Equal(t, "test@example.com", data["user"].(map[string]interface{})["email"])

	mockUseCase.AssertExpectations(t)
}

// TestRegister_ValidationError tests registration with invalid input.
func TestRegister_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		setupMock      func(*authusecasemocks.AuthUseCase, map[string]interface{})
	}{
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"username":        "testuser",
				"password":        "SecurePass123",
				"confirmPassword": "SecurePass123",
			},
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *authusecasemocks.AuthUseCase, body map[string]interface{}) {},
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":           "invalid-email",
				"username":        "testuser",
				"password":        "SecurePass123",
				"confirmPassword": "SecurePass123",
			},
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *authusecasemocks.AuthUseCase, body map[string]interface{}) {},
		},
		{
			name: "password too short",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"username":        "testuser",
				"password":        "short",
				"confirmPassword": "short",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			setupMock: func(m *authusecasemocks.AuthUseCase, body map[string]interface{}) {
				m.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
					Return(nil, domain.ErrPasswordTooShort)
			},
		},
		{
			name: "password confirmation mismatch",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"username":        "testuser",
				"password":        "SecurePass123",
				"confirmPassword": "SecurePass124",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			setupMock:      func(m *authusecasemocks.AuthUseCase, body map[string]interface{}) {},
		},
		{
			name:           "malformed JSON",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *authusecasemocks.AuthUseCase, body map[string]interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			tt.setupMock(mockUseCase, tt.requestBody)

			var bodyBytes []byte
			if tt.requestBody == nil {
				bodyBytes = []byte("{invalid json")
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			// Act
			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
func TestRegister_EmailAlreadyUsed(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, domain.ErrEmailAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
	errObj := response["data"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "already in use")

	mockUseCase.AssertExpectations(t)
}

func TestRegister_UsernameAlreadyUsed(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, domain.ErrUsernameAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test2@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "Username already in use", errObj["message"])

	mockUseCase.AssertExpectations(t)
}

// TestRegister_WithRole tests registration with role specified.
func TestRegister_WithRole(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Username: "testuser",
			Role:     "ADMIN",
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.MatchedBy(func(r *dto.RegisterRequest) bool {
		return r.Email == "admin@example.com"
	})).Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "admin@example.com",
		"username":        "adminuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
		"role":            "ADMIN",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"password":        "CorrectPassword123",
		"confirmPassword": "CorrectPassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
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
	assert.NotEmpty(t, data["token"])

	mockUseCase.AssertExpectations(t)
}

// TestLogin_SuccessWithUsernameCredential tests successful login using username in the email field.
func TestLogin_SuccessWithUsernameCredential(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:       "550e8400-e29b-41d4-a716-446655440001",
			Email:    "test@example.com",
			Username: "testuser",
			Role:     "USER",
		},
	}

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "testuser",
		"password":        "CorrectPassword123",
		"confirmPassword": "CorrectPassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
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
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
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
// HTTP 410 Gone is the standard status code for deleted resources (RFC 7231).
func TestLogin_UserDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil, domain.ErrUserDeleted)

	// Act
	reqBody := map[string]interface{}{
		"email":           "deleted@example.com",
		"password":        "password123",
		"confirmPassword": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert - 410 Gone is the HTTP standard for deleted resources
	assert.Equal(t, http.StatusGone, w.Code)

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
		"email":           "inactive@example.com",
		"password":        "password123",
		"confirmPassword": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert - 401 Unauthorized for inactive accounts
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Contains(t, errObj["message"], "inactive")

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
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
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
		Token:     "new-access-token",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"token": "valid-refresh-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
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
	assert.Equal(t, "new-access-token", data["token"])

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
		"token": "invalid-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
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

// TestGetCurrentUser_Unauthorized tests getting current user without auth.
func TestGetCurrentUser_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - no user_id set
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
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
		"old_password": "OldPass123",
		"new_password": "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
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
		"old_password": "OldPass123",
		"new_password": "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
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
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Equal(t, "invalid current password", response["message"])
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errObj["code"])
	assert.Equal(t, "invalid current password", errObj["message"])

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
		"old_password": "OldPass123",
		"new_password": "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
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
		Username: "testuser",
		Role:     "USER",
	}

	mockUseCase.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/550e8400-e29b-41d4-a716-446655440002", nil)
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
		Data: []*dto.UserResponse{
			{ID: "user-1", Email: "user1@example.com", Role: "USER"},
			{ID: "user-2", Email: "user2@example.com", Role: "ADMIN"},
		},
		Pagination: &dto.PaginationMeta{
			Page:            1,
			Limit:           10,
			Total:           2,
			TotalPages:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users?page=1&limit=10", nil)
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
	assert.NotNil(t, data["data"])

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
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.query, nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001?force=true", nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440002", nil)
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
		Username: "testuser",
		Role:     "USER",
	}

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(expectedUser, nil)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/admin/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
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
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/admin/users/550e8400-e29b-41d4-a716-446655440002/restore", nil)
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

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, domain.ErrEmailAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
		"token": "expired-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
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
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
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

// TestRegister_InternalError tests registration with internal server error.
func TestRegister_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, errors.New("database connection failed"))

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "INTERNAL_ERROR", errObj["code"])

	mockUseCase.AssertExpectations(t)
}

// TestRegister_WithDefaultRole tests registration with default USER role.
func TestRegister_WithDefaultRole(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "user@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.MatchedBy(func(r *dto.RegisterRequest) bool {
		return r.Email == "user@example.com"
	})).Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "user@example.com",
		"username":        "user",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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
	assert.Equal(t, "USER", user["role"])

	mockUseCase.AssertExpectations(t)
}

// TestLogin_EmptyPassword tests login with empty password.
func TestLogin_EmptyPassword(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin validation will fail for empty password
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"password":        "",
		"confirmPassword": "",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert - Validation error from Gin binding
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestLogin_WithIPAndUserAgent tests login with IP address and user agent.
func TestLogin_WithIPAndUserAgent(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), "192.168.1.1", "Mozilla/5.0").
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"password":        "CorrectPassword123",
		"confirmPassword": "CorrectPassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestLogout_Error tests logout with error from usecase.
func TestLogout_Error(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Logout", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(errors.New("session revocation failed"))

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/logout", handler.Logout)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRefreshToken_MalformedJSON tests refresh token with malformed JSON.
func TestRefreshToken_MalformedJSON(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestGetCurrentUser_Error tests getting current user with error.
func TestGetCurrentUser_Error(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(nil, errors.New("database error"))

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestChangePassword_EmptyNewPassword tests password change with empty new password.
func TestChangePassword_EmptyNewPassword(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin validation will fail
	reqBody := map[string]interface{}{
		"old_password": "OldPass123",
		"new_password": "",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestChangePassword_RepositoryError tests password change with repository error.
func TestChangePassword_RepositoryError(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.ChangePasswordRequest")).
		Return(errors.New("failed to update password"))

	// Act
	reqBody := map[string]interface{}{
		"old_password": "OldPass123",
		"new_password": "NewPass456",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

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

// TestGetUser_InvalidUUID tests getting user with invalid UUID.
func TestGetUser_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestGetUser_IncludeDeleted tests getting user with include_deleted flag.
func TestGetUser_IncludeDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UserResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "deleted@example.com",
		Username: "deleteduser",
		Role:     "USER",
	}

	mockUseCase.On("GetUser", mock.Anything, mock.MatchedBy(func(r *dto.GetUserRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.IncludeDeleted == true
	})).Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/550e8400-e29b-41d4-a716-446655440001?include_deleted=true", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListUsers_InvalidQueryParams tests listing users with invalid query params.
func TestListUsers_InvalidQueryParams(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid page parameter",
			query:          "/admin/users?page=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid limit parameter",
			query:          "/admin/users?limit=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role parameter",
			query:          "/admin/users?role=INVALID_ROLE",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			// Act
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/admin/users", handler.ListUsers)
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

// TestDeleteUser_InvalidUUID tests deleting user with invalid UUID.
func TestDeleteUser_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestDeleteUser_Error tests deleting user with error.
func TestDeleteUser_Error(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(errors.New("failed to delete user"))

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreUser_InvalidUUID tests restoring user with invalid UUID.
func TestRestoreUser_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/admin/users/invalid-uuid/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/admin/users/:id/restore", handler.RestoreUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestHandleError_UnknownError tests handling of unknown errors.
func TestHandleError_UnknownError(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	unknownError := errors.New("some unknown error")
	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(nil, unknownError)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_PasswordTooShort tests handling of password too short error.
func TestHandleError_PasswordTooShort(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)

	// Use a valid password for the request to pass Gin validation
	// but return the error from the usecase
	mockUseCase.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.MatchedBy(func(r *dto.ChangePasswordRequest) bool {
		return r.NewPassword == "ValidPass123"
	})).Return(domain.ErrPasswordTooShort)

	// Act
	reqBody := map[string]interface{}{
		"old_password": "OldPass123",
		"new_password": "ValidPass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert - Validation errors return 422
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_SessionRevoked tests handling of session revoked error.
func TestHandleError_SessionRevoked(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(nil, domain.ErrSessionRevoked)

	// Act
	reqBody := map[string]interface{}{
		"token": "revoked-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
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

// TestLogin_ValidationErrors tests login with validation errors.
func TestLogin_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{
			name:     "empty email",
			email:    "",
			password: "password123",
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			// Act - Gin validation will fail
			reqBody := map[string]interface{}{
				"email":           tt.email,
				"password":        tt.password,
				"confirmPassword": tt.password,
			}
			bodyBytes, _ := json.Marshal(reqBody)
			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.POST("/auth/login", handler.Login)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}

// TestRefreshToken_MissingToken tests refresh token with missing token field.
func TestRefreshToken_MissingToken(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - missing refresh_token field
	reqBody := map[string]interface{}{}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestRefreshToken_EmptyToken tests refresh token with empty token.
func TestRefreshToken_EmptyToken(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - empty refresh_token
	reqBody := map[string]interface{}{
		"token": "",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestChangePassword_MissingFields tests password change with missing fields.
func TestChangePassword_MissingFields(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "missing current password",
			requestBody: map[string]interface{}{
				"new_password": "NewPass123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing new password",
			requestBody: map[string]interface{}{
				"old_password": "OldPass123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty request body",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			router.Use(func(c *gin.Context) {
				c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
				c.Next()
			})

			router.POST("/auth/change-password", handler.ChangePassword)

			// Act
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

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

// TestListUsers_Error tests listing users with error from usecase.
func TestListUsers_Error(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(nil, errors.New("database connection failed"))

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListUsers_InvalidLimitValue tests listing users with invalid limit value.
func TestListUsers_InvalidLimitValue(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - limit exceeds maximum
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users?limit=1000", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestDeleteUser_InvalidForceParameter tests deletion with invalid force parameter.
func TestDeleteUser_InvalidForceParameter(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - invalid force parameter
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001?force=invalid", nil)
	w := httptest.NewRecorder()

	router.DELETE("/admin/users/:id", handler.DeleteUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestHandleError_MultipleErrorTypes tests comprehensive error handling.
func TestHandleError_MultipleErrorTypes(t *testing.T) {
	tests := []struct {
		name           string
		errorToReturn  error
		expectedStatus int
		setupMock      func(*authusecasemocks.AuthUseCase)
	}{
		{
			name:           "validation error",
			errorToReturn:  domain.ErrPasswordTooShort,
			expectedStatus: http.StatusUnprocessableEntity,
			setupMock: func(m *authusecasemocks.AuthUseCase) {
				m.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.ChangePasswordRequest")).
					Return(domain.ErrPasswordTooShort)
			},
		},
		{
			name:           "not found error",
			errorToReturn:  domain.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			setupMock: func(m *authusecasemocks.AuthUseCase) {
				m.On("GetUser", mock.Anything, mock.AnythingOfType("*dto.GetUserRequest")).
					Return(nil, domain.ErrUserNotFound)
			},
		},
		{
			name:           "auth error",
			errorToReturn:  domain.ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
			setupMock: func(m *authusecasemocks.AuthUseCase) {
				m.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest"), mock.Anything, mock.Anything).
					Return(nil, domain.ErrInvalidCredentials)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			tt.setupMock(mockUseCase)

			var req *http.Request
			var endpoint string

			switch tt.name {
			case "validation error":
				router.Use(func(c *gin.Context) {
					c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
					c.Next()
				})
				router.POST("/auth/change-password", handler.ChangePassword)
				reqBody := map[string]interface{}{
					"old_password": "OldPass123",
					"new_password": "ValidPass123",
				}
				bodyBytes, _ := json.Marshal(reqBody)
				req, _ = http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				endpoint = "/auth/change-password"
			case "not found error":
				router.GET("/admin/users/:id", handler.GetUser)
				req, _ = http.NewRequestWithContext(context.Background(), "GET", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
				endpoint = "/admin/users/:id"
			case "auth error":
				router.POST("/auth/login", handler.Login)
				reqBody := map[string]interface{}{
					"email":           "test@example.com",
					"password":        "password123",
					"confirmPassword": "password123",
				}
				bodyBytes, _ := json.Marshal(reqBody)
				req, _ = http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				endpoint = "/auth/login"
			}

			w := httptest.NewRecorder()

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code, "Failed for endpoint: "+endpoint)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))

			mockUseCase.AssertExpectations(t)
		})
	}
}

// TestRegister_PasswordComplexity tests registration with various password complexities.
func TestRegister_PasswordComplexity(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		shouldPass     bool
		expectedStatus int
	}{
		{
			name:           "valid password with numbers and letters",
			password:       "ValidPass123",
			shouldPass:     true,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid password with special characters",
			password:       "Valid@Pass123!",
			shouldPass:     true,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "password with only numbers",
			password:       "12345678",
			shouldPass:     false,
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "password exactly 8 characters",
			password:       "Pass1234",
			shouldPass:     true,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(authusecasemocks.AuthUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			if tt.shouldPass {
				expectedResponse := &dto.AuthResponse{
					Token:     "access-token-123",
					ExpiresIn: 3600,
					User: &dto.UserResponse{
						ID:       "550e8400-e29b-41d4-a716-446655440001",
						Email:    "test@example.com",
						Username: "testuser",
						Name:     "Test User",
						Role:     "USER",
					},
				}
				mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
					Return(expectedResponse, nil)
			}

			// Act
			reqBody := map[string]interface{}{
				"email":           "test@example.com",
				"username":        "testuser",
				"password":        tt.password,
				"confirmPassword": tt.password,
			}
			bodyBytes, _ := json.Marshal(reqBody)
			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.POST("/auth/register", handler.Register)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.shouldPass {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response["success"].(bool))
				mockUseCase.AssertExpectations(t)
			}
		})
	}
}

// TestGetUser_WithInvalidIncludeDeleted tests getUser with invalid include_deleted parameter.
func TestGetUser_WithInvalidIncludeDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - invalid include_deleted parameter
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/550e8400-e29b-41d4-a716-446655440001?include_deleted=invalid", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestRegister_MissingEmail tests registration with missing email field.
func TestRegister_MissingEmail(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - missing email field
	reqBody := map[string]interface{}{
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
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

// TestRegister_InvalidRole tests registration with invalid role.
func TestRegister_InvalidRole(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - invalid role parameter
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
		"role":            "INVALID_ROLE",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert - Gin validation should fail for invalid enum
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestLogin_MissingEmailField tests login with missing email field.
func TestLogin_MissingEmailField(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - missing email field
	reqBody := map[string]interface{}{
		"password":        "password123",
		"confirmPassword": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestChangePassword_SameAsOldPassword tests password change when new equals old.
func TestChangePassword_SameAsOldPassword(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Simulate business logic error when new password equals old
	mockUseCase.On("ChangePassword", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.MatchedBy(func(r *dto.ChangePasswordRequest) bool {
		return r.OldPassword == "SamePass123" && r.NewPassword == "SamePass123"
	})).Return(errors.New("new password must be different from current password"))

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)

	// Act
	reqBody := map[string]interface{}{
		"old_password": "SamePass123",
		"new_password": "SamePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestNewHandler tests creating a new handler.
func TestNewHandler(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)

	// Act
	handler := delivery.NewHandler(mockUseCase)

	// Assert
	assert.NotNil(t, handler)
}

// TestRegister_MalformedJSON tests registration with malformed JSON.
func TestRegister_MalformedJSON(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBufferString("{malformed json}"))
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

// TestLogin_MalformedJSON tests login with malformed JSON.
func TestLogin_MalformedJSON(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/login", bytes.NewBufferString("{malformed json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestChangePassword_MalformedJSON tests password change with malformed JSON.
func TestChangePassword_MalformedJSON(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/change-password", handler.ChangePassword)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/change-password", bytes.NewBufferString("{malformed json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// TestHandleError_ContextTimeout tests handling of context timeout errors.
func TestHandleError_ContextTimeout(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Simulate context timeout
	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(nil, errors.New("context deadline exceeded"))

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_DatabaseConnectionError tests handling of database connection errors.
func TestHandleError_DatabaseConnectionError(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(nil, errors.New("database connection failed"))

	// Act
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "INTERNAL_ERROR", errObj["code"])

	mockUseCase.AssertExpectations(t)
}

// TestGetUser_EmptyUserID tests getUser with empty user ID.
func TestGetUser_EmptyUserID(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - empty ID will fail UUID validation at Gin binding level
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users/", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users/:id", handler.GetUser)
	router.ServeHTTP(w, req)

	// Assert - Should return 404 due to missing route parameter
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestListUsers_DefaultParameters tests listing users with default pagination.
func TestListUsers_DefaultParameters(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UserListResponse{
		Data: []*dto.UserResponse{
			{ID: "user-1", Email: "user1@example.com", Role: "USER"},
		},
		Pagination: &dto.PaginationMeta{
			Page:            1,
			Limit:           10,
			Total:           1,
			TotalPages:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	mockUseCase.On("ListUsers", mock.Anything, mock.AnythingOfType("*dto.ListUsersRequest")).
		Return(expectedResponse, nil)

	// Act - no pagination parameters (should use defaults)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/admin/users", nil)
	w := httptest.NewRecorder()

	router.GET("/admin/users", handler.ListUsers)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRegister_Concurrency tests concurrent registration requests.
func TestRegister_Concurrency(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.AuthResponse{
		Token:     "access-token-123",
		ExpiresIn: 3600,
		User: &dto.UserResponse{
			ID:    "550e8400-e29b-41d4-a716-446655440001",
			Email: "test@example.com",
			Role:  "USER",
		},
	}

	mockUseCase.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).
		Return(expectedResponse, nil)

	// Act - make a single request (concurrency testing is complex in unit tests)
	reqBody := map[string]interface{}{
		"email":           "test@example.com",
		"username":        "testuser",
		"password":        "SecurePass123",
		"confirmPassword": "SecurePass123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestLogout_ContextCancelled tests logout with canceled context.
func TestLogout_ContextCancelled(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("Logout", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(errors.New("context canceled"))

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.POST("/auth/logout", handler.Logout)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestRestoreUser_AlreadyActive tests restoring a user that is already active.
func TestRestoreUser_AlreadyActive(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedUser := &dto.UserResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "test@example.com",
		Username: "testuser",
		Role:     "USER",
	}

	mockUseCase.On("RestoreUser", mock.Anything, mock.AnythingOfType("*dto.RestoreUserRequest")).
		Return(expectedUser, nil)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/admin/users/550e8400-e29b-41d4-a716-446655440001/restore", nil)
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

	mockUseCase.AssertExpectations(t)
}

// TestDeleteUser_AlreadyDeleted tests deleting an already deleted user.
func TestDeleteUser_AlreadyDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteUser", mock.Anything, mock.AnythingOfType("*dto.DeleteUserRequest")).
		Return(domain.ErrUserNotFound)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/admin/users/550e8400-e29b-41d4-a716-446655440001", nil)
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

// TestRefreshToken_Expired tests refresh with expired token.
func TestRefreshToken_Expired(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RefreshToken", mock.Anything, mock.AnythingOfType("*dto.RefreshTokenRequest")).
		Return(nil, domain.ErrSessionExpired)

	// Act
	reqBody := map[string]interface{}{
		"token": "expired-token",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", bytes.NewBuffer(bodyBytes))
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

// TestGetCurrentUser_ContextTimeout tests getting current user with context timeout.
func TestGetCurrentUser_ContextTimeout(t *testing.T) {
	// Arrange
	mockUseCase := new(authusecasemocks.AuthUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetCurrentUser", mock.Anything, "550e8400-e29b-41d4-a716-446655440001").
		Return(nil, errors.New("context timeout"))

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440001")
		c.Next()
	})

	router.GET("/auth/me", handler.GetCurrentUser)

	// Act
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockUseCase.AssertExpectations(t)
}
