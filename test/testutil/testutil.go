// Package testutil provides testing utilities for the project.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
)

// SetupTestDB creates an in-memory SQLite database for testing.
func SetupTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate all provided models
	err = db.AutoMigrate(models...)
	require.NoError(t, err)

	return db
}

// CleanupDB cleans up the database tables.
func CleanupDB(t *testing.T, db *gorm.DB, models ...interface{}) {
	for _, model := range models {
		err := db.Unscoped().Delete(model).Error
		require.NoError(t, err)
	}
}

// CreateTestUser creates a test user in the database.
func CreateTestUser(t *testing.T, db *gorm.DB, email string) *domain.User {
	user := &domain.User{
		Email:        email,
		PasswordHash: "hashed_password",
		Role:         domain.RoleUser,
		IsActive:     true,
	}

	err := db.Create(user).Error
	require.NoError(t, err)

	return user
}

// CreateTestProfile creates a test profile in the database.
func CreateTestProfile(t *testing.T, db *gorm.DB, userID string) *domain.Profile {
	profile := &domain.Profile{
		UserID:    userID,
		FirstName: "Test",
		LastName:  "User",
		Bio:       "Test user bio",
	}

	err := db.Create(profile).Error
	require.NoError(t, err)

	return profile
}

// CreateTestActivityLog creates a test activity log in the database.
func CreateTestActivityLog(t *testing.T, db *gorm.DB, userID, action string) *domain.ActivityLog {
	activity := &domain.ActivityLog{
		UserID:   userID,
		Action:   action,
		Resource: "test",
	}

	err := db.Create(activity).Error
	require.NoError(t, err)

	return activity
}

// GetTestEnv gets an environment variable for testing or returns default.
func GetTestEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WaitForCondition waits for a condition to be true or timeout.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, checkInterval time.Duration) {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			t.Fatal("Timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// WithTimeout creates a context with timeout.
func WithTimeout(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// AssertEventually asserts that a condition eventually becomes true.
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Condition was not met within %v: %s", timeout, msg)
}

// AssertNever asserts that a condition never becomes true within timeout.
func AssertNever(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			t.Fatalf("Condition should not have been true: %s", msg)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// RandomString generates a random string for testing.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
