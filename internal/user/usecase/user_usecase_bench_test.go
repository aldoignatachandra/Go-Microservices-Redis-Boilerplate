package usecase_test

import (
	"context"
	"testing"

	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	mocks "github.com/ignata/go-microservices-boilerplate/internal/user/usecase/mocks"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/stretchr/testify/mock"
)

// BenchmarkUpdateProfile benchmarks the profile update operation.
func BenchmarkUpdateProfile(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	userRepo.On("GetProfile", mock.Anything, mock.Anything).Return(getTestProfile(), nil)
	userRepo.On("UpdateProfile", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
	activityRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.UpdateProfileRequest{
		UserID:    "test-user-id",
		FirstName: strPtr("Jane"),
		LastName:  strPtr("Smith"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.UpdateProfile(context.Background(), req)
	}
}

// BenchmarkGetUser benchmarks the user retrieval operation.
func BenchmarkGetUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(getTestUser(), nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.GetUserRequest{
		ID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetUser(context.Background(), req)
	}
}

// BenchmarkActivateUser benchmarks the user activation operation.
func BenchmarkActivateUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	inactiveUser := *getTestUser()
	inactiveUser.IsActive = false
	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(&inactiveUser, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.ActivateUserRequest{
		ID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.ActivateUser(context.Background(), req)
	}
}

// BenchmarkDeactivateUser benchmarks the user deactivation operation.
func BenchmarkDeactivateUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(getTestUser(), nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.DeactivateUserRequest{
		ID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.DeactivateUser(context.Background(), req)
	}
}

// BenchmarkDeleteUser benchmarks the user deletion operation.
func BenchmarkDeleteUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	userRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.DeleteUserRequest{
		ID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uc.DeleteUser(context.Background(), req)
	}
}

// BenchmarkRestoreUser benchmarks the user restoration operation.
func BenchmarkRestoreUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventPublisher)

	userRepo.On("Restore", mock.Anything, mock.Anything).Return(nil)
	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(getTestUser(), nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("", nil)

	log, _ := logger.New(&logger.Config{Level: "info", Format: "json"})
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.RestoreUserRequest{
		ID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.RestoreUser(context.Background(), req)
	}
}
