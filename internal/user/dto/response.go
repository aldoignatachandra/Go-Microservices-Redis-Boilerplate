// Package dto provides response DTOs for the user service.
package dto

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
)

// UserResponse represents a user in responses (aligned with Bun-Hono).
type UserResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ProfileResponse represents a user profile in responses (aligned with Bun-Hono).
type ProfileResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FromProfile creates a ProfileResponse from a domain.Profile.
func FromProfile(profile *domain.Profile) *ProfileResponse {
	if profile == nil {
		return nil
	}

	return &ProfileResponse{
		ID:   profile.ID,
		Name: profile.Name,
	}
}

// FromUser creates a UserResponse from a domain.User.
func FromUser(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}

	var deletedAt *time.Time
	if user.DeletedAt.Valid {
		deletedAt = &user.DeletedAt.Time
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Name:      user.Name,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		DeletedAt: deletedAt,
	}
}

// UserListResponse represents a list of users (aligned with Bun-Hono).
type UserListResponse struct {
	Data       []*UserResponse `json:"data"`
	Pagination *PaginationMeta `json:"meta"`
}

// PaginationMeta contains pagination metadata (aligned with Bun-Hono).
type PaginationMeta struct {
	Page            int   `json:"page"`
	Limit           int   `json:"limit"`
	Total           int64 `json:"total"`
	TotalPages      int   `json:"totalPages"`
	HasNextPage     bool  `json:"hasNextPage"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}

// FromUserList creates a UserListResponse from a domain.UserList.
func FromUserList(list *domain.UserList) *UserListResponse {
	if list == nil {
		return &UserListResponse{}
	}

	users := make([]*UserResponse, len(list.Users))
	for i, user := range list.Users {
		users[i] = FromUser(user)
	}

	return &UserListResponse{
		Data: users,
		Pagination: &PaginationMeta{
			Page:            list.Page,
			Limit:           list.Limit,
			Total:           list.Total,
			TotalPages:      list.TotalPages,
			HasNextPage:     list.Page < list.TotalPages,
			HasPreviousPage: list.Page > 1,
		},
	}
}

// ActivityLogResponse represents an activity log in responses.
type ActivityLogResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"userId"`
	Action    string                 `json:"action"`
	Entity    string                 `json:"entity,omitempty"`
	EntityID  string                 `json:"entityId,omitempty"`
	IPAddress string                 `json:"ipAddress,omitempty"`
	UserAgent string                 `json:"userAgent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

// FromActivityLog creates an ActivityLogResponse from a domain.ActivityLog.
func FromActivityLog(log *domain.ActivityLog) *ActivityLogResponse {
	if log == nil {
		return nil
	}

	return &ActivityLogResponse{
		ID:        log.ID,
		UserID:    log.UserID,
		Action:    log.Action,
		Entity:    log.Entity,
		EntityID:  log.EntityID,
		IPAddress: log.IPAddress,
		UserAgent: log.UserAgent,
		Details:   log.Details,
		CreatedAt: log.CreatedAt,
	}
}

// ActivityLogListResponse represents a list of activity logs.
type ActivityLogListResponse struct {
	Data       []*ActivityLogResponse `json:"data"`
	Pagination *PaginationMeta        `json:"meta"`
}

// FromActivityLogList creates an ActivityLogListResponse from a domain.ActivityLogList.
func FromActivityLogList(list *domain.ActivityLogList) *ActivityLogListResponse {
	if list == nil {
		return &ActivityLogListResponse{}
	}

	logs := make([]*ActivityLogResponse, len(list.Logs))
	for i, log := range list.Logs {
		logs[i] = FromActivityLog(log)
	}

	return &ActivityLogListResponse{
		Data: logs,
		Pagination: &PaginationMeta{
			Page:            list.Page,
			Limit:           list.Limit,
			Total:           list.Total,
			TotalPages:      list.TotalPages,
			HasNextPage:     list.Page < list.TotalPages,
			HasPreviousPage: list.Page > 1,
		},
	}
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// DeleteResponse represents a delete operation response.
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RestoreResponse represents a restore operation response.
type RestoreResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	User    *UserResponse `json:"user,omitempty"`
}
