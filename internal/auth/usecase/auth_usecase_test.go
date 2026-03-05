// Package usecase provides tests for the auth use case.
package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// --- Mock Repositories ---

// MockUserRepository is a mock for repository.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) HardDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Restore(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.User, error) {
	args := m.Called(ctx, id, opts)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string, opts *domain.ParanoidOptions) (*domain.User, error) {
	args := m.Called(ctx, email, opts)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindAll(ctx context.Context, req *dto.ListUsersRequest) (*domain.UserList, error) {
	args := m.Called(ctx, req)
	if list, ok := args.Get(0).(*domain.UserList); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

// MockSessionRepository is a mock for repository.SessionRepository.
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	args := m.Called(ctx, refreshToken)
	if session, ok := args.Get(0).(*domain.Session); ok {
		return session, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if sessions, ok := args.Get(0).([]*domain.Session); ok {
		return sessions, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSessionRepository) Revoke(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Helper ---

func newTestAuthUseCase(userRepo *MockUserRepository, sessionRepo *MockSessionRepository) usecase.AuthUseCase {
	return usecase.NewAuthUseCase(
		userRepo,
		sessionRepo,
		(*eventbus.Producer)(nil), // nil event bus for unit tests
		usecase.Config{
			JWTSecret:        "test-secret-key-at-least-32-chars-long!!",
			JWTExpiresIn:     time.Hour,
			RefreshExpiresIn: 7 * 24 * time.Hour,
			BcryptCost:       4, // low cost for fast tests
			ServiceName:      "auth-service-test",
		},
	)
}

// --- Tests ---

func TestRegister_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	req := &dto.RegisterRequest{
		Email:    "test@example.com",
		Password: "SecureP@ss123",
	}

	// Email does not exist
	userRepo.On("ExistsByEmail", mock.Anything, req.Email).Return(false, nil)
	// User creation succeeds
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	// Session creation succeeds
	sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Session")).Return(nil)

	response, err := uc.Register(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, "Bearer", response.TokenType)
	assert.NotNil(t, response.User)

	userRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	req := &dto.RegisterRequest{
		Email:    "existing@example.com",
		Password: "SecureP@ss123",
	}

	// Email already exists
	userRepo.On("ExistsByEmail", mock.Anything, req.Email).Return(true, nil)

	response, err := uc.Register(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrEmailAlreadyUsed, err)

	userRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	// We need a real bcrypt hash to test password verification
	// Using cost 4 for fast test
	testUser := &domain.User{
		Model: domain.Model{
			ID: "test-user-id",
		},
		Email:        "test@example.com",
		PasswordHash: "$2a$04$test", // Will not match; test structure only
		Role:         domain.RoleUser,
		IsActive:     true,
	}

	req := &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong-password",
	}

	userRepo.On("FindByEmail", mock.Anything, req.Email, mock.Anything).Return(testUser, nil)

	_, err := uc.Login(context.Background(), req, "127.0.0.1", "test-agent")

	// This should fail because the password hash doesn't match
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidCredentials, err)

	userRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	req := &dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password",
	}

	userRepo.On("FindByEmail", mock.Anything, req.Email, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound)

	_, err := uc.Login(context.Background(), req, "127.0.0.1", "test-agent")

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidCredentials, err) // Should map to invalid credentials

	userRepo.AssertExpectations(t)
}

func TestLogin_InactiveUser(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	testUser := &domain.User{
		Model:    domain.Model{ID: "test-inactive-user"},
		Email:    "inactive@example.com",
		IsActive: false, // Inactive
		Role:     domain.RoleUser,
	}

	req := &dto.LoginRequest{
		Email:    "inactive@example.com",
		Password: "password",
	}

	userRepo.On("FindByEmail", mock.Anything, req.Email, mock.Anything).Return(testUser, nil)

	_, err := uc.Login(context.Background(), req, "127.0.0.1", "test-agent")

	assert.Error(t, err)
	assert.Equal(t, domain.ErrUserInactive, err)

	userRepo.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	userID := "test-user-id"

	sessionRepo.On("RevokeAllForUser", mock.Anything, userID).Return(nil)

	err := uc.Logout(context.Background(), userID)

	assert.NoError(t, err)
	sessionRepo.AssertExpectations(t)
}

func TestGetCurrentUser_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	testUser := &domain.User{
		Model:    domain.Model{ID: "test-user-id"},
		Email:    "test@example.com",
		Role:     domain.RoleUser,
		IsActive: true,
	}

	userRepo.On("FindByID", mock.Anything, "test-user-id", mock.Anything).Return(testUser, nil)

	response, err := uc.GetCurrentUser(context.Background(), "test-user-id")

	assert.NoError(t, err)
	assert.NotNil(t, response)

	userRepo.AssertExpectations(t)
}

func TestGetCurrentUser_NotFound(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	userRepo.On("FindByID", mock.Anything, "nonexistent-id", mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound)

	response, err := uc.GetCurrentUser(context.Background(), "nonexistent-id")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrUserNotFound, err)

	userRepo.AssertExpectations(t)
}

func TestDeleteUser_SoftDelete(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	testUser := &domain.User{
		Model:    domain.Model{ID: "delete-me"},
		Email:    "delete@example.com",
		Role:     domain.RoleUser,
		IsActive: true,
	}

	req := &dto.DeleteUserRequest{
		ID:    "delete-me",
		Force: false,
	}

	userRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testUser, nil)
	userRepo.On("Delete", mock.Anything, req.ID).Return(nil)
	sessionRepo.On("RevokeAllForUser", mock.Anything, req.ID).Return(nil)

	err := uc.DeleteUser(context.Background(), req)

	assert.NoError(t, err)
	userRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestDeleteUser_HardDelete(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	testUser := &domain.User{
		Model:    domain.Model{ID: "hard-delete-me"},
		Email:    "hard-delete@example.com",
		Role:     domain.RoleUser,
		IsActive: true,
	}

	req := &dto.DeleteUserRequest{
		ID:    "hard-delete-me",
		Force: true,
	}

	userRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testUser, nil)
	userRepo.On("HardDelete", mock.Anything, req.ID).Return(nil)
	sessionRepo.On("RevokeAllForUser", mock.Anything, req.ID).Return(nil)

	err := uc.DeleteUser(context.Background(), req)

	assert.NoError(t, err)
	userRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestRestoreUser_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	uc := newTestAuthUseCase(userRepo, sessionRepo)

	restoredUser := &domain.User{
		Model:    domain.Model{ID: "restored-user"},
		Email:    "restored@example.com",
		Role:     domain.RoleUser,
		IsActive: true,
	}

	req := &dto.RestoreUserRequest{
		ID: "restored-user",
	}

	userRepo.On("Restore", mock.Anything, req.ID).Return(nil)
	userRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(restoredUser, nil)

	response, err := uc.RestoreUser(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, response)

	userRepo.AssertExpectations(t)
}
