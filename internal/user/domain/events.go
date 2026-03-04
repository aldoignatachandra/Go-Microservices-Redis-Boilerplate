// Package domain provides event types for the user service.
package domain

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// Event types for the user service.
const (
	// User events (consumed from auth service)
	EventUserCreated   = "user.created"
	EventUserUpdated   = "user.updated"
	EventUserDeleted   = "user.deleted"
	EventUserRestored  = "user.restored"
	EventUserLoggedIn  = "user.logged_in"
	EventUserLoggedOut = "user.logged_out"
	EventUserActivated   = "user.activated"
	EventUserDeactivated = "user.deactivated"

	// Profile events
	EventProfileUpdated = "profile.updated"

	// Activity events (published by user service)
	EventActivityCreated = "activity.created"
)

// Activity types for logging user actions.
const (
	ActivityProfileUpdated = "profile_updated"
	ActivityUserActivated   = "user_activated"
	ActivityUserDeactivated = "user_deactivated"
	ActivityUserDeleted      = "user_deleted"
	ActivityUserRestored      = "user_restored"
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

// NewUserEvent creates a new user event from raw data.
func NewUserEvent(eventType, userID string, data map[string]interface{}) *UserEvent {
	event := &UserEvent{
		EventType: eventType,
		UserID:    userID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}

	if email, ok := data["email"].(string); ok {
		event.Email = email
	}
	if role, ok := data["role"].(string); ok {
		event.Role = role
	}

	return event
}

// ActivityEvent represents an activity event for logging.
type ActivityEvent struct {
	EventType  string                 `json:"event_type"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewActivityEvent creates a new activity event.
func NewActivityEvent(userID, action string) *ActivityEvent {
	return &ActivityEvent{
		EventType: EventActivityCreated,
		UserID:    userID,
		Action:    action,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithResource adds resource information to the activity event.
func (e *ActivityEvent) WithResource(resource, resourceID string) *ActivityEvent {
	e.Resource = resource
	e.ResourceID = resourceID
	return e
}

// WithRequestInfo adds request information to the activity event.
func (e *ActivityEvent) WithRequestInfo(ipAddress, userAgent string) *ActivityEvent {
	e.IPAddress = ipAddress
	e.UserAgent = userAgent
	return e
}

// WithMetadata adds metadata to the activity event.
func (e *ActivityEvent) WithMetadata(key string, value interface{}) *ActivityEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// ToMap converts the activity event to a map for Redis storage.
func (e *ActivityEvent) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"event_type": e.EventType,
		"user_id":    e.UserID,
		"action":     e.Action,
		"timestamp":  e.Timestamp.UnixMilli(),
	}

	if e.Resource != "" {
		result["resource"] = e.Resource
	}
	if e.ResourceID != "" {
		result["resource_id"] = e.ResourceID
	}
	if e.IPAddress != "" {
		result["ip_address"] = e.IPAddress
	}
	if e.UserAgent != "" {
		result["user_agent"] = e.UserAgent
	}
	if e.Metadata != nil {
		result["metadata"] = e.Metadata
	}

	return result
}

// ToActivityLog converts the activity event to an ActivityLog entity.
func (e *ActivityEvent) ToActivityLog() *ActivityLog {
	log := &ActivityLog{
		UserID:     e.UserID,
		Action:     e.Action,
		Resource:   e.Resource,
		ResourceID: e.ResourceID,
		IPAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		Metadata:   e.Metadata,
	}
	return log
}

// NewProfileUpdatedEvent creates a new profile updated event.
func NewProfileUpdatedEvent(userID string, profile *Profile) *eventbus.Event {
	return eventbus.NewEvent(EventProfileUpdated, "user-service", map[string]interface{}{
		"user_id":     userID,
		"first_name":  profile.FirstName,
		"last_name":   profile.LastName,
		"avatar":      profile.Avatar,
		"bio":         profile.Bio,
	})
}

// NewUserActivatedEvent creates a new user activated event.
func NewUserActivatedEvent(userID, email string) *eventbus.Event {
	return eventbus.NewEvent(EventUserActivated, "user-service", map[string]interface{}{
		"user_id": userID,
		"email":   email,
	})
}

// NewUserDeactivatedEvent creates a new user deactivated event.
func NewUserDeactivatedEvent(userID, email string) *eventbus.Event {
	return eventbus.NewEvent(EventUserDeactivated, "user-service", map[string]interface{}{
		"user_id": userID,
		"email":   email,
	})
}

// NewUserDeletedEvent creates a new user deleted event.
func NewUserDeletedEvent(userID string) *eventbus.Event {
	return eventbus.NewEvent(EventUserDeleted, "user-service", map[string]interface{}{
		"user_id": userID,
	})
}

// NewUserRestoredEvent creates a new user restored event.
func NewUserRestoredEvent(userID, email string) *eventbus.Event {
	return eventbus.NewEvent(EventUserRestored, "user-service", map[string]interface{}{
		"user_id": userID,
		"email":   email,
	})
}
