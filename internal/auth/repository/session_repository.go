// Package repository provides session data access.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"gorm.io/gorm"
)

// SessionRepository defines the interface for session data access.
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *domain.Session) error
	// FindByRefreshToken finds a session by refresh token
	FindByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	// FindByUserID finds all sessions for a user
	FindByUserID(ctx context.Context, userID string) ([]*domain.Session, error)
	// Revoke revokes a session
	Revoke(ctx context.Context, id string) error
	// RevokeAllForUser revokes all sessions for a user
	RevokeAllForUser(ctx context.Context, userID string) error
	// DeleteByUserID deletes all sessions for a user (hard delete for single session policy)
	DeleteByUserID(ctx context.Context, userID string) error
	// DeleteExpired deletes all expired sessions
	DeleteExpired(ctx context.Context) error
}

// gormSessionRepository implements SessionRepository using GORM.
type gormSessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository.
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &gormSessionRepository{db: db}
}

// Create creates a new session.
func (r *gormSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	result := r.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		return fmt.Errorf("failed to create session: %w", result.Error)
	}
	return nil
}

// FindByRefreshToken finds a session by refresh token.
func (r *gormSessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	var session domain.Session
	result := r.db.WithContext(ctx).
		Where("token = ?", refreshToken).
		First(&session)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", result.Error)
	}

	return &session, nil
}

// FindByUserID finds all sessions for a user.
func (r *gormSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > NOW()", userID).
		Order("created_at DESC").
		Find(&sessions)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", result.Error)
	}

	return sessions, nil
}

// Revoke revokes a session.
func (r *gormSessionRepository) Revoke(ctx context.Context, id string) error {
	now := timeNow()
	result := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke session: %w", result.Error)
	}

	return nil
}

// RevokeAllForUser revokes all sessions for a user.
func (r *gormSessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	now := timeNow()
	result := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke sessions: %w", result.Error)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a user (hard delete for single session policy).
func (r *gormSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&domain.Session{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete sessions: %w", result.Error)
	}

	return nil
}

// DeleteExpired deletes all expired sessions.
func (r *gormSessionRepository) DeleteExpired(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < NOW() OR revoked_at IS NOT NULL").
		Delete(&domain.Session{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", result.Error)
	}

	return nil
}

// timeNow is a variable for testing.
var timeNow = func() time.Time {
	return time.Now().UTC()
}
