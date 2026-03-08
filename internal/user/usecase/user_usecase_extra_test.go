package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase/mocks"
)

func TestListUsers_Success(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ListUsersRequest{
		Page:  1,
		Limit: 10,
	}

	userList := &domain.UserList{
		Users: []*domain.User{
			{
				Model: domain.Model{ID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Email: "test1@example.com",
			},
		},
		Page:       1,
		Limit:      10,
		Total:      1,
		TotalPages: 1,
	}

	mockUserRepo.On("FindAll", mock.Anything, req).
		Return(userList, nil)

	resp, err := uc.ListUsers(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Users))
	mockUserRepo.AssertExpectations(t)
}

func TestListUsers_Error(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ListUsersRequest{
		Page:  1,
		Limit: 10,
	}

	mockUserRepo.On("FindAll", mock.Anything, req).
		Return(nil, errors.New("db error"))

	resp, err := uc.ListUsers(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUserRepo.AssertExpectations(t)
}

func TestGetActivityLogs_Success(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ListActivityLogsRequest{
		UserID: "user-123",
		Page:   1,
		Limit:  10,
	}

	logList := &domain.ActivityLogList{
		Logs: []*domain.ActivityLog{
			{
				Model:  domain.Model{ID: "log-1"},
				UserID: "user-123",
				Action: "login",
			},
		},
		Page:       1,
		Limit:      10,
		Total:      1,
		TotalPages: 1,
	}

	mockActivityRepo.On("FindAll", mock.Anything, req).
		Return(logList, nil)

	resp, err := uc.GetActivityLogs(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Logs))
	mockActivityRepo.AssertExpectations(t)
}

func TestGetActivityLogs_Error(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ListActivityLogsRequest{
		UserID: "user-123",
		Page:   1,
		Limit:  10,
	}

	mockActivityRepo.On("FindAll", mock.Anything, req).
		Return(nil, errors.New("db error"))

	resp, err := uc.GetActivityLogs(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockActivityRepo.AssertExpectations(t)
}

func TestUpdateProfile_UpdateError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.UpdateProfileRequest{
		UserID:    "user-1",
		FirstName: strPtr("Jane"),
	}

	mockUserRepo.On("GetProfile", mock.Anything, req.UserID).Return(&domain.Profile{UserID: req.UserID}, nil)
	mockUserRepo.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*domain.Profile")).Return(errors.New("update error"))

	err := uc.UpdateProfile(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update profile")
	mockUserRepo.AssertExpectations(t)
}

func TestActivateUser_RepoError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ActivateUserRequest{ID: "user-1"}

	mockUserRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("db error"))

	err := uc.ActivateUser(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
	mockUserRepo.AssertExpectations(t)
}

func TestActivateUser_UpdateError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.ActivateUserRequest{ID: "user-1"}
	user := &domain.User{}

	mockUserRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(user, nil)
	mockUserRepo.On("Restore", mock.Anything, req.ID).Return(errors.New("restore error"))

	err := uc.ActivateUser(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to activate user")
}

func TestDeactivateUser_UpdateError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.DeactivateUserRequest{ID: "user-1"}
	user := &domain.User{}

	mockUserRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(user, nil)
	mockUserRepo.On("Delete", mock.Anything, req.ID).Return(errors.New("delete error"))
	mockEventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	err := uc.DeactivateUser(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deactivate user")
}

func TestDeleteUser_HardDeleteError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.DeleteUserRequest{ID: "user-1", Force: true}

	mockUserRepo.On("HardDelete", mock.Anything, req.ID).Return(errors.New("db error"))

	err := uc.DeleteUser(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hard delete user")
}

func TestDeleteUser_SoftDeleteError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.DeleteUserRequest{ID: "user-1", Force: false}

	mockUserRepo.On("Delete", mock.Anything, req.ID).Return(errors.New("db error"))

	err := uc.DeleteUser(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user")
}

func TestRestoreUser_FindByIDError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.RestoreUserRequest{ID: "user-1"}

	mockUserRepo.On("Restore", mock.Anything, req.ID).Return(nil)
	mockUserRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("db error"))

	resp, err := uc.RestoreUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get restored user")
}

func TestLogActivity_Error(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.LogActivityRequest{
		UserID:   "user-1",
		Action:   "test",
		Resource: "test-resource",
	}

	mockActivityRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := uc.LogActivity(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log activity")
}

func TestUpdateProfile_GetProfileError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.UpdateProfileRequest{
		UserID:    "user-1",
		FirstName: strPtr("Jane"),
	}

	mockUserRepo.On("GetProfile", mock.Anything, req.UserID).Return(nil, errors.New("db error"))

	err := uc.UpdateProfile(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get profile")
}

func TestGetUser_RepoError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.GetUserRequest{ID: "user-1"}

	mockUserRepo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("db error"))

	resp, err := uc.GetUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get user")
}

func TestRestoreUser_RestoreError(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockEventBus := new(MockEventPublisher)
	logger := zap.NewNop()
	uc := usecase.NewUserUseCase(mockUserRepo, mockActivityRepo, mockEventBus, logger)

	req := &dto.RestoreUserRequest{ID: "user-1"}

	mockUserRepo.On("Restore", mock.Anything, req.ID).Return(errors.New("db error"))

	resp, err := uc.RestoreUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to restore user")
}
