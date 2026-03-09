// Package dto provides Data Transfer Objects for the auth service.
package dto

import (
	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/validator"
)

// RegisterRequest represents a user registration request (aligned with Bun-Hono).
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"omitempty,max=255"`
}

// Validate validates the registration request.
func (r *RegisterRequest) Validate() error {
	if err := validator.ValidateUsername(r.Username); err != nil {
		return err
	}
	return validator.ValidatePassword(r.Password)
}

// LoginRequest represents a login request (aligned with Bun-Hono).
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a token refresh request (aligned with Bun-Hono).
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// UpdateUserRequest represents a user update request.
type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email,max=255"`
	Password string `json:"password" binding:"omitempty"`
	Name     string `json:"name" binding:"omitempty,max=255"`
}

// Validate validates the update user request.
func (r *UpdateUserRequest) Validate() error {
	if r.Password == "" {
		return nil
	}
	return validator.ValidatePassword(r.Password)
}

// ChangePasswordRequest represents a password change request (aligned with Bun-Hono).
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// Validate validates the change password request.
func (r *ChangePasswordRequest) Validate() error {
	return validator.ValidatePassword(r.NewPassword)
}

// UpdateRoleRequest represents a role update request.
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=ADMIN USER"`
}

// ListUsersRequest represents a request to list users.
type ListUsersRequest struct {
	Page           int    `form:"page" binding:"omitempty,min=1"`
	Limit          int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Role           string `form:"role" binding:"omitempty,oneof=ADMIN USER"`
	Search         string `form:"search" binding:"omitempty"`
	IncludeDeleted bool   `form:"include_deleted"`
	OnlyDeleted    bool   `form:"only_deleted"`
}

// GetPage returns the page number with default.
func (r *ListUsersRequest) GetPage() int {
	if r.Page < 1 {
		return 1
	}
	return r.Page
}

// GetLimit returns the limit with default.
func (r *ListUsersRequest) GetLimit() int {
	if r.Limit < 1 {
		return 10
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

// GetParanoidOptions returns paranoid options from the request.
func (r *ListUsersRequest) GetParanoidOptions() *domain.ParanoidOptions {
	return &domain.ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyDeleted:    r.OnlyDeleted,
		OnlyActive:     !r.IncludeDeleted && !r.OnlyDeleted,
	}
}

// GetUserRequest represents a request to get a user.
type GetUserRequest struct {
	ID             string `uri:"id" binding:"required,uuid"`
	IncludeDeleted bool   `form:"include_deleted"`
}

// GetParanoidOptions returns paranoid options from the request.
func (r *GetUserRequest) GetParanoidOptions() *domain.ParanoidOptions {
	return &domain.ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyActive:     !r.IncludeDeleted,
	}
}

// DeleteUserRequest represents a request to delete a user.
type DeleteUserRequest struct {
	ID    string `uri:"id" binding:"required,uuid"`
	Force bool   `form:"force"`
}

// RestoreUserRequest represents a request to restore a deleted user.
type RestoreUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}
