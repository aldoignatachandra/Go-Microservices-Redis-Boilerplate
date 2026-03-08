// Package domain provides domain entities for the auth service.
package domain

import (
	"time"

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

// IsValid checks if the role is valid.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleUser
}

// Model is the base model for all entities.
type Model struct {
	ID        string         `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// User represents a user entity.
type User struct {
	Model
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Username     string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Name         string     `gorm:"type:varchar(255)" json:"name"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Role         Role       `gorm:"type:varchar(50);not null;default:'USER'" json:"role"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// TableName specifies the table name for User.
func (User) TableName() string {
	return "users"
}

// IsAdmin checks if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// CanLogin checks if the user can login.
func (u *User) CanLogin() bool {
	return !u.DeletedAt.Valid
}

// TouchLastLogin updates the last login timestamp.
func (u *User) TouchLastLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
}

// BeforeCreate is a GORM hook that runs before creating a user.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	// Set default role if not specified
	if u.Role == "" {
		u.Role = RoleUser
	}

	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a user.
func (u *User) BeforeUpdate(_ *gorm.DB) error {
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeUser returns a copy of the user without sensitive fields.
func (u *User) ToSafeUser() *SafeUser {
	return &SafeUser{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// SafeUser represents a user without sensitive fields.
type SafeUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserList represents a list of users with pagination info.
type UserList struct {
	Users      []*User `json:"users"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
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
	// Default to only active
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
