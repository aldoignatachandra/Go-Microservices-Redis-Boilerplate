// Package dto provides response DTOs for the auth service.
package dto

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
)

// AuthResponse represents an authentication response (aligned with Bun-Hono).
type AuthResponse struct {
	Token        string        `json:"token"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	ExpiresIn    int64         `json:"expires_in,omitempty"`
	User         *UserResponse `json:"user"`
}

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

// PaginationMeta contains pagination metadata.
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
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
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
