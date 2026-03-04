// Package domain provides event types for the auth service.
package domain

import "time"

// Event types for the auth service.
const (
	// User events
	EventUserCreated   = "user.created"
	EventUserUpdated   = "user.updated"
	EventUserDeleted   = "user.deleted"
	EventUserRestored  = "user.restored"

	// Authentication events
	EventUserLoggedIn  = "user.logged_in"
	EventUserLoggedOut = "user.logged_out"
	EventUserRefreshedToken = "user.refreshed_token"
)

// UserEvent represents a user-related event.
type UserEvent struct {
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewUserCreatedEvent creates a new user created event.
func NewUserCreatedEvent(user *User) *UserEvent {
	return &UserEvent{
		EventType: EventUserCreated,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewUserUpdatedEvent creates a new user updated event.
func NewUserUpdatedEvent(user *User) *UserEvent {
	return &UserEvent{
		EventType: EventUserUpdated,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewUserDeletedEvent creates a new user deleted event.
func NewUserDeletedEvent(userID string) *UserEvent {
	return &UserEvent{
		EventType: EventUserDeleted,
		UserID:    userID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewUserRestoredEvent creates a new user restored event.
func NewUserRestoredEvent(user *User) *UserEvent {
	return &UserEvent{
		EventType: EventUserRestored,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewUserLoggedInEvent creates a new user logged in event.
func NewUserLoggedInEvent(user *User, ipAddress, userAgent string) *UserEvent {
	return &UserEvent{
		EventType: EventUserLoggedIn,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Timestamp: time.Now().UTC(),
		Metadata: map[string]interface{}{
			"ip_address": ipAddress,
			"user_agent": userAgent,
		},
	}
}

// NewUserLoggedOutEvent creates a new user logged out event.
func NewUserLoggedOutEvent(userID string) *UserEvent {
	return &UserEvent{
		EventType: EventUserLoggedOut,
		UserID:    userID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithMetadata adds metadata to the event.
func (e *UserEvent) WithMetadata(key string, value interface{}) *UserEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// ToMap converts the event to a map for Redis storage.
func (e *UserEvent) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"event_type": e.EventType,
		"user_id":    e.UserID,
		"timestamp":  e.Timestamp.UnixMilli(),
	}

	if e.Email != "" {
		result["email"] = e.Email
	}
	if e.Role != "" {
		result["role"] = e.Role
	}
	if e.Metadata != nil {
		result["metadata"] = e.Metadata
	}

	return result
}
