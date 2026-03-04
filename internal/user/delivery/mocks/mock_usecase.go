// Package mocks provides mock implementations for testing.
package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
)

// MockUserUseCase is a mock implementation of UserUseCase.
type MockUserUseCase struct {
	mock.Mock
}

func (m *MockUserUseCase) UpdateProfile(ctx interface{}, req *dto.UpdateProfileRequest) error {
	args := m.Called(ctx, mock.AnythingOfType("*dto.UpdateProfileRequest"))
	return args.Error(0)
}

func (m *MockUserUseCase) GetProfile(ctx interface{}, req *dto.GetUserRequest) (*dto.ProfileResponse, error) {
	args := m.Called(ctx, mock.AnythingOfType("*dto.GetUserRequest"))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProfileResponse), args.Error(1)
}

func (m *MockUserUseCase) GetUser(ctx interface{}, req *dto.GetUserRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, mock.AnythingOfType("*dto.GetUserRequest"))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserResponse), args.Error(1)
}

func (m *MockUserUseCase) ListUsers(ctx interface{}, req *dto.ListUsersRequest) (*dto.UserListResponse, error) {
	args := m.Called(ctx, mock.AnythingOfType("*dto.ListUsersRequest"))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserListResponse), args.Error(1)
}

func (m *MockUserUseCase) ActivateUser(ctx interface{}, req *dto.ActivateUserRequest) error {
	args := m.Called(ctx, mock.AnythingOfType("*dto.ActivateUserRequest"))
	return args.Error(0)
}

func (m *MockUserUseCase) DeactivateUser(ctx interface{}, req *dto.DeactivateUserRequest) error {
	args := m.Called(ctx, mock.AnythingOfType("*dto.DeactivateUserRequest"))
	return args.Error(0)
}

func (m *MockUserUseCase) DeleteUser(ctx interface{}, req *dto.DeleteUserRequest) error {
	args := m.Called(ctx, mock.AnythingOfType("*dto.DeleteUserRequest"))
	return args.Error(0)
}

func (m *MockUserUseCase) RestoreUser(ctx interface{}, req *dto.RestoreUserRequest) (*dto.RestoreResponse, error) {
	args := m.Called(ctx, mock.AnythingOfType("*dto.RestoreUserRequest"))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RestoreResponse), args.Error(1)
}

func (m *MockUserUseCase) LogActivity(ctx interface{}, req *dto.LogActivityRequest) error {
	args := m.Called(ctx, mock.AnythingOfType("*dto.LogActivityRequest"))
	return args.Error(0)
}

func (m *MockUserUseCase) GetActivityLogs(ctx interface{}, req *dto.ListActivityLogsRequest) (*dto.ActivityLogListResponse, error) {
	args := m.Called(ctx, mock.AnythingOfType("*dto.ListActivityLogsRequest"))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ActivityLogListResponse), args.Error(1)
}
