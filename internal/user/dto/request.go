// Package dto provides Data Transfer Objects for the user service.
package dto

import (
	"errors"
)

// GetUserRequest represents a request to get a user.
type GetUserRequest struct {
	ID             string `uri:"id" binding:"required,uuid"`
	IncludeDeleted bool   `form:"include_deleted"`
}

// Validate validates the get user request.
func (r *GetUserRequest) Validate() error {
	if r.ID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

// GetParanoidOptions returns paranoid options from the request.
func (r *GetUserRequest) GetParanoidOptions() *ParanoidOptions {
	return &ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyActive:     !r.IncludeDeleted,
	}
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

// Validate validates the list users request.
func (r *ListUsersRequest) Validate() error {
	return nil // No validation needed, binding handles it
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
func (r *ListUsersRequest) GetParanoidOptions() *ParanoidOptions {
	return &ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyDeleted:    r.OnlyDeleted,
		OnlyActive:     !r.IncludeDeleted && !r.OnlyDeleted,
	}
}

// DeleteUserRequest represents a request to delete a user.
type DeleteUserRequest struct {
	ID    string `uri:"id" binding:"required,uuid"`
	Force bool   `form:"force"`
}

// Validate validates the delete user request.
func (r *DeleteUserRequest) Validate() error {
	if r.ID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

// RestoreUserRequest represents a request to restore a deleted user.
type RestoreUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// Validate validates the restore user request.
func (r *RestoreUserRequest) Validate() error {
	if r.ID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

// ActivateUserRequest represents a request to activate a user.
type ActivateUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// Validate validates the activate user request.
func (r *ActivateUserRequest) Validate() error {
	if r.ID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

// DeactivateUserRequest represents a request to deactivate a user.
type DeactivateUserRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// Validate validates the deactivate user request.
func (r *DeactivateUserRequest) Validate() error {
	if r.ID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

// LogActivityRequest represents a request to log user activity.
type LogActivityRequest struct {
	UserID   string `json:"user_id" binding:"required,uuid"`
	Action   string `json:"action" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Details  string `json:"details,omitempty"`
}

// Validate validates the log activity request.
func (r *LogActivityRequest) Validate() error {
	if r.UserID == "" {
		return errors.New("user ID is required")
	}
	if r.Action == "" {
		return errors.New("action is required")
	}
	if r.Resource == "" {
		return errors.New("resource is required")
	}
	return nil
}

// ListActivityLogsRequest represents a request to list activity logs.
type ListActivityLogsRequest struct {
	UserID   string `form:"user_id" binding:"omitempty,uuid"`
	Action   string `form:"action" binding:"omitempty"`
	Resource string `form:"resource" binding:"omitempty"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// Validate validates the list activity logs request.
func (r *ListActivityLogsRequest) Validate() error {
	return nil // No validation needed, binding handles it
}

// GetPage returns the page number with default.
func (r *ListActivityLogsRequest) GetPage() int {
	if r.Page < 1 {
		return 1
	}
	return r.Page
}

// GetLimit returns the limit with default.
func (r *ListActivityLogsRequest) GetLimit() int {
	if r.Limit < 1 {
		return 20
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

// ParanoidOptions defines options for querying with soft delete support.
type ParanoidOptions struct {
	IncludeDeleted bool `form:"include_deleted" json:"include_deleted"`
	OnlyDeleted    bool `form:"only_deleted" json:"only_deleted"`
	OnlyActive     bool `form:"only_active" json:"only_active"`
}

// DefaultParanoidOptions returns default paranoid options (only active).
func DefaultParanoidOptions() *ParanoidOptions {
	return &ParanoidOptions{
		OnlyActive: true,
	}
}

// Validate validates the paranoid options.
func (p *ParanoidOptions) Validate() error {
	if !p.IncludeDeleted && !p.OnlyDeleted && !p.OnlyActive {
		p.OnlyActive = true
	}
	return nil
}

// ShouldIncludeDeleted returns true if deleted records should be included.
func (p *ParanoidOptions) ShouldIncludeDeleted() bool {
	return p.IncludeDeleted || p.OnlyDeleted
}

// ShouldOnlyDeleted returns true if only deleted records should be returned.
func (p *ParanoidOptions) ShouldOnlyDeleted() bool {
	return p.OnlyDeleted
}
