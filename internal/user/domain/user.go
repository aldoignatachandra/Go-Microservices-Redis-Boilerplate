// Package domain provides domain entities for the user service.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents the user role enum.
type Role string

const (
	// RoleAdmin represents an administrator role.
	RoleAdmin Role = "ADMIN"
	// RoleUser represents a standard user role.
	RoleUser Role = "USER"
)

// Model is the base model for all entities.
type Model struct {
	ID        string         `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate is a GORM hook that sets the UUID.
func (m *Model) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// User represents a user entity.
type User struct {
	Model
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Role         Role       `gorm:"type:varchar(50);not null;default:'USER'" json:"role"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	Profile      *Profile   `gorm:"foreignKey:UserID" json:"profile,omitempty"`
}

// IsAdmin checks if user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// CanLogin checks if user can login.
func (u *User) CanLogin() bool {
	return u.IsActive && !u.DeletedAt.Valid
}

// TouchLastLogin updates the last login timestamp.
func (u *User) TouchLastLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
}

// Profile represents user profile information.
type Profile struct {
	Model
	UserID    string `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	FirstName string `gorm:"type:varchar(100)" json:"first_name,omitempty"`
	LastName  string `gorm:"type:varchar(100)" json:"last_name,omitempty"`
	Avatar    string `gorm:"type:text" json:"avatar,omitempty"`
	Bio       string `gorm:"type:text" json:"bio,omitempty"`
}

// TableName specifies the table name for User.
func (User) TableName() string {
	return "users"
}

// TableName specifies the table name for Profile.
func (Profile) TableName() string {
	return "profiles"
}

// FullName returns the user's full name.
func (p *Profile) FullName() string {
	if p.FirstName == "" && p.LastName == "" {
		return ""
	}
	return p.FirstName + " " + p.LastName
}

// ActivityLog represents an activity log entry.
type ActivityLog struct {
	Model
	UserID     string                 `gorm:"type:uuid;not null;index" json:"user_id"`
	Action     string                 `gorm:"type:varchar(100);not null" json:"action"`
	Resource   string                 `gorm:"type:varchar(100)" json:"resource,omitempty"`
	ResourceID string                 `gorm:"type:uuid" json:"resource_id,omitempty"`
	IPAddress  string                 `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent  string                 `gorm:"type:text" json:"user_agent,omitempty"`
	Metadata   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

// TableName specifies the table name for ActivityLog.
func (ActivityLog) TableName() string {
	return "activity_logs"
}

// NewActivityLog creates a new activity log entry.
func NewActivityLog(userID, action, resource, resourceID string) *ActivityLog {
	return &ActivityLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
	}
}

// WithMetadata adds metadata to the activity log.
func (a *ActivityLog) WithMetadata(key string, value interface{}) *ActivityLog {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
	return a
}

// WithRequestInfo adds request information to the activity log.
func (a *ActivityLog) WithRequestInfo(ipAddress, userAgent string) *ActivityLog {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
	return a
}

// WithDetails adds details to the activity log metadata.
func (a *ActivityLog) WithDetails(details string) *ActivityLog {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata["details"] = details
	return a
}

// UserList represents a list of users with pagination info.
type UserList struct {
	Users      []*User `json:"users"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}

// ActivityLogList represents a list of activity logs.
type ActivityLogList struct {
	Logs       []*ActivityLog `json:"logs"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}
