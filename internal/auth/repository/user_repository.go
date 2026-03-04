// Package repository provides data access for the auth service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error
	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error
	// Delete soft deletes a user
	Delete(ctx context.Context, id string) error
	// HardDelete permanently deletes a user
	HardDelete(ctx context.Context, id string) error
	// Restore restores a soft-deleted user
	Restore(ctx context.Context, id string) error
	// FindByID finds a user by ID
	FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.User, error)
	// FindByEmail finds a user by email
	FindByEmail(ctx context.Context, email string, opts *domain.ParanoidOptions) (*domain.User, error)
	// FindAll finds all users with pagination
	FindAll(ctx context.Context, req *dto.ListUsersRequest) (*domain.UserList, error)
	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// gormUserRepository implements UserRepository using GORM.
type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create creates a new user.
func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrEmailAlreadyUsed
		}
		return fmt.Errorf("failed to create user: %w", result.Error)
	}
	return nil
}

// Update updates an existing user.
func (r *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Delete soft deletes a user.
func (r *gormUserRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// HardDelete permanently deletes a user.
func (r *gormUserRepository) HardDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Restore restores a soft-deleted user.
func (r *gormUserRepository) Restore(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Unscoped().
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)

	if result.Error != nil {
		return fmt.Errorf("failed to restore user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// FindByID finds a user by ID.
func (r *gormUserRepository) FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.User, error) {
	if opts == nil {
		opts = domain.DefaultParanoidOptions()
	}

	query := r.db.WithContext(ctx)

	if opts.ShouldIncludeDeleted() {
		query = query.Unscoped()
		if opts.ShouldOnlyDeleted() {
			query = query.Where("deleted_at IS NOT NULL")
		}
	}

	var user domain.User
	result := query.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", result.Error)
	}

	return &user, nil
}

// FindByEmail finds a user by email.
func (r *gormUserRepository) FindByEmail(ctx context.Context, email string, opts *domain.ParanoidOptions) (*domain.User, error) {
	if opts == nil {
		opts = domain.DefaultParanoidOptions()
	}

	query := r.db.WithContext(ctx)

	if opts.ShouldIncludeDeleted() {
		query = query.Unscoped()
		if opts.ShouldOnlyDeleted() {
			query = query.Where("deleted_at IS NOT NULL")
		}
	}

	var user domain.User
	result := query.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", result.Error)
	}

	return &user, nil
}

// FindAll finds all users with pagination.
func (r *gormUserRepository) FindAll(ctx context.Context, req *dto.ListUsersRequest) (*domain.UserList, error) {
	if req == nil {
		req = &dto.ListUsersRequest{}
	}

	page := req.GetPage()
	limit := req.GetLimit()
	opts := req.GetParanoidOptions()

	query := r.db.WithContext(ctx).Model(&domain.User{})

	// Apply paranoid options
	if opts.ShouldIncludeDeleted() {
		query = query.Unscoped()
		if opts.ShouldOnlyDeleted() {
			query = query.Where("deleted_at IS NOT NULL")
		}
	}

	// Apply filters
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}
	if req.Search != "" {
		query = query.Where("email ILIKE ?", "%"+req.Search+"%")
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Get paginated results
	var users []*domain.User
	offset := (page - 1) * limit
	result := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find users: %w", result.Error)
	}

	return &domain.UserList{
		Users:      users,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// ExistsByEmail checks if a user exists by email.
func (r *gormUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("email = ?", email).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check user existence: %w", result.Error)
	}

	return count > 0, nil
}
