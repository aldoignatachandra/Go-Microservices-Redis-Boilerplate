// Package dto provides response DTOs for the auth service.
package dto

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
)

// AuthResponse represents an authentication response.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
	TokenType    string       `json:"token_type"`
	User         *UserResponse `json:"user"`
}

// UserResponse represents a user in responses.
type UserResponse struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// FromUser creates a UserResponse from a domain.User.
func FromUser(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}

	resp := &UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       string(user.Role),
		IsActive:   user.IsActive,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}

	if user.DeletedAt.Valid {
		resp.DeletedAt = &user.DeletedAt.Time
	}

	return resp
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

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

// ErrorDetails contains error details.
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TokenResponse represents a token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
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
