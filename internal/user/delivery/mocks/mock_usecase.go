// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
)

// MockUserUseCase is a mock implementation of UserUseCase.
type MockUserUseCase struct {
	mock.Mock
}

func (m *MockUserUseCase) UpdateProfile(ctx context.Context, req *dto.UpdateProfileRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserUseCase) GetProfile(ctx context.Context, req *dto.GetUserRequest) (*dto.ProfileResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProfileResponse), args.Error(1)
}

func (m *MockUserUseCase) GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserResponse), args.Error(1)
}

func (m *MockUserUseCase) ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.UserListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserListResponse), args.Error(1)
}

func (m *MockUserUseCase) ActivateUser(ctx context.Context, req *dto.ActivateUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserUseCase) DeactivateUser(ctx context.Context, req *dto.DeactivateUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserUseCase) DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserUseCase) RestoreUser(ctx context.Context, req *dto.RestoreUserRequest) (*dto.RestoreResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RestoreResponse), args.Error(1)
}

func (m *MockUserUseCase) LogActivity(ctx context.Context, req *dto.LogActivityRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserUseCase) GetActivityLogs(ctx context.Context, req *dto.ListActivityLogsRequest) (*dto.ActivityLogListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ActivityLogListResponse), args.Error(1)
}
