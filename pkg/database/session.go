// Package database provides database connection management and query helpers.
package database

import "context"

// HasActiveSession returns true when the user has at least one active session.
func (db *PostgresDB) HasActiveSession(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM user_sessions
			WHERE user_id = ?
			  AND revoked_at IS NULL
			  AND expires_at > NOW()
			  AND deleted_at IS NULL
		)
	`, userID).Scan(&exists).Error
	if err != nil {
		return false, err
	}

	return exists, nil
}

// HasActiveSessionByID returns true when the specific session is active for the user.
func (db *PostgresDB) HasActiveSessionByID(ctx context.Context, userID, sessionID string) (bool, error) {
	var exists bool
	err := db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM user_sessions
			WHERE id = ?
			  AND user_id = ?
			  AND revoked_at IS NULL
			  AND expires_at > NOW()
			  AND deleted_at IS NULL
		)
	`, sessionID, userID).Scan(&exists).Error
	if err != nil {
		return false, err
	}

	return exists, nil
}
