// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository.
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

func (m *MockUserRepository) FindByID(ctx context.Context, id string, opts *dto.ParanoidOptions) (*domain.User, error) {
	args := m.Called(ctx, id, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string, opts *dto.ParanoidOptions) (*domain.User, error) {
	args := m.Called(ctx, email, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindAll(ctx context.Context, req *dto.ListUsersRequest) (*domain.UserList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserList), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *MockUserRepository) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Profile), args.Error(1)
}

// MockActivityRepository is a mock implementation of ActivityRepository.
type MockActivityRepository struct {
	mock.Mock
}

func (m *MockActivityRepository) Create(ctx context.Context, log *domain.ActivityLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockActivityRepository) FindByUserID(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ActivityLogList), args.Error(1)
}

func (m *MockActivityRepository) FindAll(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ActivityLogList), args.Error(1)
}

func (m *MockActivityRepository) DeleteOlderThan(ctx context.Context, days int) error {
	args := m.Called(ctx, days)
	return args.Error(0)
}
