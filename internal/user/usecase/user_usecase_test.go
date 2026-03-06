package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	mocks "github.com/ignata/go-microservices-boilerplate/internal/user/usecase/mocks"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
)

// MockEventPublisher is a mock implementation of eventbus.EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, stream string, event *eventbus.Event) (string, error) {
	args := m.Called(ctx, stream, event)
	return args.String(0), args.Error(1)
}

// Test data
func getTestUser() *domain.User {
	return &domain.User{
		Model: domain.Model{
			ID:        "test-user-id",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Email:    "test@example.com",
		Role:     domain.RoleUser,
		IsActive: true,
	}
}

func getTestProfile() *domain.Profile {
	return &domain.Profile{
		Model: domain.Model{
			ID:        "test-profile-id",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:    "test-user-id",
		FirstName: "John",
		LastName:  "Doe",
		Bio:       "Test user",
	}
}

// TestUpdateProfile tests the UpdateProfile use case.
func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.UpdateProfileRequest
		setupMocks  func(*mocks.MockUserRepository, *mocks.MockActivityRepository, *MockEventPublisher)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful update existing profile",
			req: &dto.UpdateProfileRequest{
				UserID:    "test-user-id",
				FirstName: strPtr("Jane"),
				LastName:  strPtr("Smith"),
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, activityRepo *mocks.MockActivityRepository, eventBus *MockEventPublisher) {
				userRepo.On("GetProfile", mock.Anything, "test-user-id").Return(getTestProfile(), nil)
				userRepo.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*domain.Profile")).Return(nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
				activityRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ActivityLog")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "successful create new profile",
			req: &dto.UpdateProfileRequest{
				UserID:    "test-user-id",
				FirstName: strPtr("Jane"),
				LastName:  strPtr("Smith"),
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, activityRepo *mocks.MockActivityRepository, eventBus *MockEventPublisher) {
				userRepo.On("GetProfile", mock.Anything, "test-user-id").Return(nil, nil)
				userRepo.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*domain.Profile")).Return(nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
				activityRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ActivityLog")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "validation error - missing user ID",
			req: &dto.UpdateProfileRequest{
				FirstName: strPtr("Jane"),
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, activityRepo *mocks.MockActivityRepository, eventBus *MockEventPublisher) {
				// No mocks should be called due to validation error
			},
			wantErr:     true,
			expectedErr: domain.ErrValidationError,
		},
		{
			name: "repository error",
			req: &dto.UpdateProfileRequest{
				UserID:    "test-user-id",
				FirstName: strPtr("Jane"),
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, activityRepo *mocks.MockActivityRepository, eventBus *MockEventPublisher) {
				userRepo.On("GetProfile", mock.Anything, "test-user-id").Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			// Setup expectations
			tt.setupMocks(userRepo, activityRepo, eventBus)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			err := uc.UpdateProfile(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr) || errors.Is(err, domain.ErrValidationError))
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			activityRepo.AssertExpectations(t)
			eventBus.AssertExpectations(t)
		})
	}
}

// TestGetUser tests the GetUser use case.
func TestGetUser(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.GetUserRequest
		setupMocks  func(*mocks.MockUserRepository)
		wantErr     bool
		expectedErr error
		checkResult func(*testing.T, *dto.UserResponse)
	}{
		{
			name: "successful get user",
			req: &dto.GetUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(getTestUser(), nil)
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *dto.UserResponse) {
				assert.Equal(t, "test-user-id", resp.ID)
				assert.Equal(t, "test@example.com", resp.Email)
			},
		},
		{
			name: "user not found",
			req: &dto.GetUserRequest{
				ID: "non-existent-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				userRepo.On("FindByID", mock.Anything, "non-existent-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(nil, domain.ErrUserNotFound)
			},
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name: "validation error - missing user ID",
			req: &dto.GetUserRequest{
				ID: "",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				// No mocks should be called
			},
			wantErr:     true,
			expectedErr: domain.ErrValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			resp, err := uc.GetUser(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr) || errors.Is(err, domain.ErrValidationError))
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResult != nil {
					tt.checkResult(t, resp)
				}
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
		})
	}
}

// TestActivateUser tests the ActivateUser use case.
func TestActivateUser(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.ActivateUserRequest
		setupMocks  func(*mocks.MockUserRepository, *MockEventPublisher)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful activation",
			req: &dto.ActivateUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				inactiveUser := *getTestUser()
				inactiveUser.IsActive = false
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(&inactiveUser, nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
			},
			wantErr: false,
		},
		{
			name: "user already active",
			req: &dto.ActivateUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(getTestUser(), nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: &dto.ActivateUserRequest{
				ID: "non-existent-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("FindByID", mock.Anything, "non-existent-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(nil, domain.ErrUserNotFound)
			},
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo, eventBus)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			err := uc.ActivateUser(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			eventBus.AssertExpectations(t)
		})
	}
}

// TestDeactivateUser tests the DeactivateUser use case.
func TestDeactivateUser(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.DeactivateUserRequest
		setupMocks  func(*mocks.MockUserRepository, *MockEventPublisher)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful deactivation",
			req: &dto.DeactivateUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(getTestUser(), nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
			},
			wantErr: false,
		},
		{
			name: "user already inactive",
			req: &dto.DeactivateUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				inactiveUser := *getTestUser()
				inactiveUser.IsActive = false
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(&inactiveUser, nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: &dto.DeactivateUserRequest{
				ID: "non-existent-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("FindByID", mock.Anything, "non-existent-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(nil, domain.ErrUserNotFound)
			},
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo, eventBus)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			err := uc.DeactivateUser(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			eventBus.AssertExpectations(t)
		})
	}
}

// TestDeleteUser tests the DeleteUser use case.
func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.DeleteUserRequest
		setupMocks  func(*mocks.MockUserRepository, *MockEventPublisher)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful deletion",
			req: &dto.DeleteUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("Delete", mock.Anything, "test-user-id").Return(nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: &dto.DeleteUserRequest{
				ID: "non-existent-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("Delete", mock.Anything, "non-existent-id").Return(domain.ErrUserNotFound)
			},
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo, eventBus)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			err := uc.DeleteUser(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			eventBus.AssertExpectations(t)
		})
	}
}

// TestRestoreUser tests the RestoreUser use case.
func TestRestoreUser(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.RestoreUserRequest
		setupMocks  func(*mocks.MockUserRepository, *MockEventPublisher)
		wantErr     bool
		checkResult func(*testing.T, *dto.RestoreResponse)
	}{
		{
			name: "successful restoration",
			req: &dto.RestoreUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("Restore", mock.Anything, "test-user-id").Return(nil)
				userRepo.On("FindByID", mock.Anything, "test-user-id", mock.AnythingOfType("*dto.ParanoidOptions")).Return(getTestUser(), nil)
				eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *dto.RestoreResponse) {
				assert.True(t, resp.Success)
				assert.NotNil(t, resp.User)
				assert.Equal(t, "test-user-id", resp.User.ID)
			},
		},
		{
			name: "user not found",
			req: &dto.RestoreUserRequest{
				ID: "non-existent-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, eventBus *MockEventPublisher) {
				userRepo.On("Restore", mock.Anything, "non-existent-id").Return(domain.ErrUserNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo, eventBus)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			resp, err := uc.RestoreUser(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResult != nil {
					tt.checkResult(t, resp)
				}
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			eventBus.AssertExpectations(t)
		})
	}
}

// TestGetProfile tests the GetProfile use case.
func TestGetProfile(t *testing.T) {
	tests := []struct {
		name        string
		req         *dto.GetUserRequest
		setupMocks  func(*mocks.MockUserRepository)
		wantErr     bool
		expectedErr error
		checkResult func(*testing.T, *dto.ProfileResponse)
	}{
		{
			name: "successful get profile",
			req: &dto.GetUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				userRepo.On("GetProfile", mock.Anything, "test-user-id").Return(getTestProfile(), nil)
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *dto.ProfileResponse) {
				assert.Equal(t, "test-profile-id", resp.ID)
				assert.Equal(t, "John", resp.FirstName)
				assert.Equal(t, "Doe", resp.LastName)
			},
		},
		{
			name: "profile not found",
			req: &dto.GetUserRequest{
				ID: "test-user-id",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				userRepo.On("GetProfile", mock.Anything, "test-user-id").Return(nil, nil)
			},
			wantErr:     true,
			expectedErr: domain.ErrProfileNotFound,
		},
		{
			name: "validation error - missing user ID",
			req: &dto.GetUserRequest{
				ID: "",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository) {
				// No mocks should be called
			},
			wantErr:     true,
			expectedErr: domain.ErrValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(userRepo)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			resp, err := uc.GetProfile(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr) || errors.Is(err, domain.ErrValidationError))
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResult != nil {
					tt.checkResult(t, resp)
				}
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
		})
	}
}

// TestLogActivity tests the LogActivity use case.
func TestLogActivity(t *testing.T) {
	tests := []struct {
		name       string
		req        *dto.LogActivityRequest
		setupMocks func(*mocks.MockActivityRepository)
		wantErr    bool
	}{
		{
			name: "successful log activity",
			req: &dto.LogActivityRequest{
				UserID:   "test-user-id",
				Action:   "login",
				Resource: "auth",
			},
			setupMocks: func(activityRepo *mocks.MockActivityRepository) {
				activityRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ActivityLog")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "validation error - missing user ID",
			req: &dto.LogActivityRequest{
				Action:   "login",
				Resource: "auth",
			},
			setupMocks: func(activityRepo *mocks.MockActivityRepository) {
				// No mocks should be called
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := new(mocks.MockUserRepository)
			activityRepo := new(mocks.MockActivityRepository)
			eventBus := new(MockEventPublisher)

			tt.setupMocks(activityRepo)

			// Create use case
			log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
			uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

			// Execute
			err := uc.LogActivity(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			activityRepo.AssertExpectations(t)
		})
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}
