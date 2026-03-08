// Package domain provides session entities for the auth service.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session represents a user session.
type Session struct {
	ID         string         `gorm:"type:uuid;primary_key;" json:"id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Token      string         `gorm:"type:text;not null" json:"-"`
	ExpiresAt  time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt  time.Time      `gorm:"not null" json:"created_at"`
	RevokedAt  *time.Time     `json:"revoked_at,omitempty"`
	LastUsedAt time.Time      `gorm:"not null" json:"last_used_at"`
	DeviceType string         `gorm:"type:varchar(50)" json:"device_type"`
	UserAgent  string         `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress  string         `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for Session.
func (Session) TableName() string {
	return "sessions"
}

// IsExpired checks if the session is expired.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsRevoked checks if the session is revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsValid checks if the session is valid (not expired and not revoked).
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// Revoke revokes the session.
func (s *Session) Revoke() {
	now := time.Now().UTC()
	s.RevokedAt = &now
}

// UpdateLastUsed updates the last used timestamp.
func (s *Session) UpdateLastUsed() {
	s.LastUsedAt = time.Now().UTC()
}

// BeforeCreate is a GORM hook that runs before creating a session.
func (s *Session) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	s.CreatedAt = now
	s.LastUsedAt = now
	return nil
}

// SessionList represents a list of sessions.
type SessionList struct {
	Sessions []*Session `json:"sessions"`
	Total    int64      `json:"total"`
}
