// Package dto provides Data Transfer Objects for the auth service.
package dto

import "github.com/ignata/go-microservices-boilerplate/internal/auth/domain"

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"omitempty,oneof=ADMIN USER"`
}

// ToRole converts the role string to domain.Role.
func (r *RegisterRequest) ToRole() domain.Role {
	if r.Role == string(domain.RoleAdmin) {
		return domain.RoleAdmin
	}
	return domain.RoleUser
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateUserRequest represents a user update request.
type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=8"`
	IsActive *bool  `json:"is_active" binding:"omitempty"`
}

// ChangePasswordRequest represents a password change request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
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
	Force bool   `form:"force"` // Force hard delete
}

// RestoreUserRequest represents a request to restore a deleted user.
type RestoreUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}
