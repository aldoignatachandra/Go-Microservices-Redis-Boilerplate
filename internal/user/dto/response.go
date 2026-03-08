// Package dto provides response DTOs for the user service.
package dto

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
)

// UserResponse represents a user in responses.
type UserResponse struct {
	ID          string           `json:"id"`
	Email       string           `json:"email"`
	Username    string           `json:"username"`
	Name        string           `json:"name"`
	Role        string           `json:"role"`
	Profile     *ProfileResponse `json:"profile,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   *time.Time       `json:"deleted_at,omitempty"`
	LastLoginAt *time.Time       `json:"last_login_at,omitempty"`
}

// ProfileResponse represents a user profile in responses.
type ProfileResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	FullName  string `json:"full_name,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

// FromUser creates a UserResponse from a domain.User.
func FromUser(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}

	resp := &UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		Name:        user.Name,
		Role:        string(user.Role),
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}

	if user.DeletedAt.Valid {
		resp.DeletedAt = &user.DeletedAt.Time
	}

	if user.Profile != nil {
		resp.Profile = FromProfile(user.Profile)
	}

	return resp
}

// FromProfile creates a ProfileResponse from a domain.Profile.
func FromProfile(profile *domain.Profile) *ProfileResponse {
	if profile == nil {
		return nil
	}

	return &ProfileResponse{
		ID:        profile.ID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		FullName:  profile.FullName(),
		Avatar:    profile.Avatar,
		Bio:       profile.Bio,
	}
}

// UserListResponse represents a list of users.
type UserListResponse struct {
	Users      []*UserResponse `json:"users"`
	Pagination *PaginationMeta `json:"pagination"`
}

// PaginationMeta contains pagination metadata.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
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
		Users: users,
		Pagination: &PaginationMeta{
			Page:       list.Page,
			Limit:      list.Limit,
			Total:      list.Total,
			TotalPages: list.TotalPages,
			HasNext:    list.Page < list.TotalPages,
			HasPrev:    list.Page > 1,
		},
	}
}

// ActivityLogResponse represents an activity log in responses.
type ActivityLogResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Entity    string                 `json:"entity,omitempty"`
	EntityID  string                 `json:"entity_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
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
	Logs       []*ActivityLogResponse `json:"logs"`
	Pagination *PaginationMeta        `json:"pagination"`
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
		Logs: logs,
		Pagination: &PaginationMeta{
			Page:       list.Page,
			Limit:      list.Limit,
			Total:      list.Total,
			TotalPages: list.TotalPages,
			HasNext:    list.Page < list.TotalPages,
			HasPrev:    list.Page > 1,
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
