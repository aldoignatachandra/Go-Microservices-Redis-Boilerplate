package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	mocks "github.com/ignata/go-microservices-boilerplate/internal/user/usecase/mocks"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
)

// BenchmarkUpdateProfile benchmarks the profile update operation.
func BenchmarkUpdateProfile(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	userRepo.On("GetProfile", mock.Anything, mock.Anything).Return(testProfile, nil)
	userRepo.On("UpdateProfile", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	activityRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.UpdateProfileRequest{
		UserID:    "test-user-id",
		FirstName: strPtr("Jane"),
		LastName:  strPtr("Smith"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.UpdateProfile(context.Background(), req)
	}
}

// BenchmarkGetUser benchmarks the user retrieval operation.
func BenchmarkGetUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(testUser, nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.GetUserRequest{
		UserID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = usecase.GetUser(context.Background(), req)
	}
}

// BenchmarkActivateUser benchmarks the user activation operation.
func BenchmarkActivateUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	inactiveUser := *testUser
	inactiveUser.IsActive = false
	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(&inactiveUser, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.ActivateUserRequest{
		UserID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.ActivateUser(context.Background(), req)
	}
}

// BenchmarkDeactivateUser benchmarks the user deactivation operation.
func BenchmarkDeactivateUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(testUser, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.DeactivateUserRequest{
		UserID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.DeactivateUser(context.Background(), req)
	}
}

// BenchmarkDeleteUser benchmarks the user deletion operation.
func BenchmarkDeleteUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	userRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.DeleteUserRequest{
		UserID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.DeleteUser(context.Background(), req)
	}
}

// BenchmarkRestoreUser benchmarks the user restoration operation.
func BenchmarkRestoreUser(b *testing.B) {
	userRepo := new(mocks.MockUserRepository)
	activityRepo := new(mocks.MockActivityRepository)
	eventBus := new(MockEventBus)

	userRepo.On("Restore", mock.Anything, mock.Anything).Return(nil)
	userRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).Return(testUser, nil)
	eventBus.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	log := logger.NewLogger("info", "production")
	usecase := NewUserUseCase(userRepo, activityRepo, eventBus, log)

	req := &dto.RestoreUserRequest{
		UserID: "test-user-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = usecase.RestoreUser(context.Background(), req)
	}
}
