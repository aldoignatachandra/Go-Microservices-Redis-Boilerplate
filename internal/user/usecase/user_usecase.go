// Package usecase provides business logic for the user service.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/repository"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// UserUseCase defines the interface for user business logic.
type UserUseCase interface {
	// User queries
	GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.UserResponse, error)
	ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.UserListResponse, error)

	// User commands
	ActivateUser(ctx context.Context, req *dto.ActivateUserRequest) error
	DeactivateUser(ctx context.Context, req *dto.DeactivateUserRequest) error
	DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) error
	RestoreUser(ctx context.Context, req *dto.RestoreUserRequest) (*dto.RestoreResponse, error)

	// Activity operations
	LogActivity(ctx context.Context, req *dto.LogActivityRequest) error
	GetActivityLogs(ctx context.Context, req *dto.ListActivityLogsRequest) (*dto.ActivityLogListResponse, error)
}

// userUseCase implements UserUseCase.
type userUseCase struct {
	userRepo     repository.UserRepository
	activityRepo repository.ActivityRepository
	eventBus     eventbus.EventPublisher
	logger       *zap.Logger
}

// NewUserUseCase creates a new user use case.
func NewUserUseCase(
	userRepo repository.UserRepository,
	activityRepo repository.ActivityRepository,
	eventBus eventbus.EventPublisher,
	log *zap.Logger,
) UserUseCase {
	return &userUseCase{
		userRepo:     userRepo,
		activityRepo: activityRepo,
		eventBus:     eventBus,
		logger:       log,
	}
}

// GetUser retrieves a user by ID.
func (uc *userUseCase) GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.UserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	opts := dto.DefaultParanoidOptions()
	if req.IncludeDeleted {
		opts.IncludeDeleted = true
	}

	user, err := uc.userRepo.FindByID(ctx, req.ID, opts)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dto.FromUser(user), nil
}

// ListUsers retrieves a paginated list of users.
func (uc *userUseCase) ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.UserListResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	list, err := uc.userRepo.FindAll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return dto.FromUserList(list), nil
}

// ActivateUser activates a user account (restores soft-deleted user).
func (uc *userUseCase) ActivateUser(ctx context.Context, req *dto.ActivateUserRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	_, err := uc.userRepo.FindByID(ctx, req.ID, &dto.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	restoreReq := &dto.RestoreUserRequest{ID: req.ID}
	_, err = uc.RestoreUser(ctx, restoreReq)
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	return nil
}

// DeactivateUser deactivates a user account (soft deletes user).
func (uc *userUseCase) DeactivateUser(ctx context.Context, req *dto.DeactivateUserRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	_, err := uc.userRepo.FindByID(ctx, req.ID, dto.DefaultParanoidOptions())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	deleteReq := &dto.DeleteUserRequest{ID: req.ID, Force: false}
	if err := uc.DeleteUser(ctx, deleteReq); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user.
func (uc *userUseCase) DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	if req.Force {
		if err := uc.userRepo.HardDelete(ctx, req.ID); err != nil {
			return fmt.Errorf("failed to hard delete user: %w", err)
		}
	} else {
		if err := uc.userRepo.Delete(ctx, req.ID); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}
	}

	// Publish event
	event := domain.NewUserDeletedEvent(req.ID)
	utils.ApplyRequestMetadataToEvent(ctx, event)
	if _, err := uc.eventBus.Publish(ctx, eventbus.StreamUserEvents, event); err != nil {
		uc.logger.Error("failed to publish user deleted event", zap.Error(err))
	}

	return nil
}

// RestoreUser restores a soft-deleted user.
func (uc *userUseCase) RestoreUser(ctx context.Context, req *dto.RestoreUserRequest) (*dto.RestoreResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	if err := uc.userRepo.Restore(ctx, req.ID); err != nil {
		return nil, fmt.Errorf("failed to restore user: %w", err)
	}

	// Get restored user
	user, err := uc.userRepo.FindByID(ctx, req.ID, &dto.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get restored user: %w", err)
	}

	// Publish event
	event := domain.NewUserRestoredEvent(req.ID, user.Email)
	utils.ApplyRequestMetadataToEvent(ctx, event)
	if _, err := uc.eventBus.Publish(ctx, eventbus.StreamUserEvents, event); err != nil {
		uc.logger.Error("failed to publish user restored event", zap.Error(err))
	}

	return &dto.RestoreResponse{
		Success: true,
		Message: "User restored successfully",
		User:    dto.FromUser(user),
	}, nil
}

// LogActivity logs user activity.
func (uc *userUseCase) LogActivity(ctx context.Context, req *dto.LogActivityRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	activity := domain.NewActivityLog(req.UserID, req.Action, req.Resource, req.UserID).
		WithRequestInfo(req.IPAddress, req.UserAgent).
		WithDetails(req.Details)

	if err := uc.activityRepo.Create(ctx, activity); err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}

	return nil
}

// GetActivityLogs retrieves activity logs.
func (uc *userUseCase) GetActivityLogs(ctx context.Context, req *dto.ListActivityLogsRequest) (*dto.ActivityLogListResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidationError, err)
	}

	list, err := uc.activityRepo.FindAll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity logs: %w", err)
	}

	return dto.FromActivityLogList(list), nil
}
