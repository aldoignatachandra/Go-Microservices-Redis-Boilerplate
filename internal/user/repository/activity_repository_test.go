// Package repository_test provides tests for the activity repository.
package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/google/uuid"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/repository"
)

// TestActivityRepository_Create tests creating activity logs.
func TestActivityRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Auto migrate ActivityLog
	err := db.AutoMigrate(&domain.ActivityLog{})
	require.NoError(t, err)

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		log := &domain.ActivityLog{
			Model: domain.Model{
				ID:        uuid.New().String(),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			UserID:    uuid.New().String(),
			Action:    "user.login",
			Resource:  "auth",
			Metadata:  map[string]interface{}{"ip": "192.168.1.1"},
		}

		err := repo.Create(ctx, log)
		assert.NoError(t, err)
		assert.NotEmpty(t, log.ID)
	})

	t.Run("create with nil db context", func(t *testing.T) {
		log := &domain.ActivityLog{
			UserID:   uuid.New().String(),
			Action:   "user.logout",
			Resource: "auth",
		}

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := repo.Create(ctx, log)
		assert.Error(t, err)
	})
}

// TestActivityRepository_FindByUserID tests finding activity logs by user ID.
func TestActivityRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	err := db.AutoMigrate(&domain.ActivityLog{})
	require.NoError(t, err)

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	// Create test user and logs
	userID := uuid.New().String()

	// Create multiple activity logs
	for i := 0; i < 5; i++ {
		log := &domain.ActivityLog{
			Model: domain.Model{
				ID:        uuid.New().String(),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			UserID:   userID,
			Action:   fmt.Sprintf("action.%d", i),
			Resource: "test",
		}
		err := db.Create(log).Error
		require.NoError(t, err)
	}

	t.Run("find all logs for user", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID: userID,
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindByUserID(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Logs, 5)
		assert.Equal(t, 5, result.Total)
	})

	t.Run("find with pagination", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID: userID,
			Page:   1,
			Limit:  2,
		}

		result, err := repo.FindByUserID(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 2)
		assert.Equal(t, 5, result.Total)
		assert.Equal(t, 3, result.TotalPages)
	})

	t.Run("find with action filter", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID:  userID,
			Action:  "action.1",
			Page:    1,
			Limit:   10,
		}

		result, err := repo.FindByUserID(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 1)
	})

	t.Run("find with resource filter", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID:   userID,
			Resource: "test",
			Page:     1,
			Limit:    10,
		}

		result, err := repo.FindByUserID(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 5)
	})

	t.Run("find for non-existent user", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID: uuid.New().String(),
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindByUserID(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 0)
		assert.Equal(t, 0, result.Total)
	})

	t.Run("nil request defaults", func(t *testing.T) {
		result, err := repo.FindByUserID(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestActivityRepository_FindAll tests finding all activity logs.
func TestActivityRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	err := db.AutoMigrate(&domain.ActivityLog{})
	require.NoError(t, err)

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	// Create test logs for multiple users
	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	for i := 0; i < 3; i++ {
		log1 := &domain.ActivityLog{
			Model:    domain.Model{ID: uuid.New().String(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
			UserID:   userID1,
			Action:   "user.action",
			Resource: "resource",
		}
		err := db.Create(log1).Error
		require.NoError(t, err)

		log2 := &domain.ActivityLog{
			Model:    domain.Model{ID: uuid.New().String(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
			UserID:   userID2,
			Action:   "admin.action",
			Resource: "admin",
		}
		err = db.Create(log2).Error
		require.NoError(t, err)
	}

	t.Run("find all logs", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			Page:  1,
			Limit: 10,
		}

		result, err := repo.FindAll(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 6)
	})

	t.Run("find with user filter", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID: userID1,
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindAll(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 3)
	})

	t.Run("find with action filter", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			Action: "user.action",
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindAll(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 3)
	})

	t.Run("find with resource filter", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			Resource: "admin",
			Page:     1,
			Limit:    10,
		}

		result, err := repo.FindAll(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 3)
	})

	t.Run("find with multiple filters", func(t *testing.T) {
		req := &dto.ListActivityLogsRequest{
			UserID:   userID1,
			Action:   "user.action",
			Resource: "resource",
			Page:     1,
			Limit:    10,
		}

		result, err := repo.FindAll(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, result.Logs, 3)
	})

	t.Run("nil request defaults", func(t *testing.T) {
		result, err := repo.FindAll(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestActivityRepository_DeleteOlderThan tests deleting old activity logs.
func TestActivityRepository_DeleteOlderThan(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	err := db.AutoMigrate(&domain.ActivityLog{})
	require.NoError(t, err)

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	// Create old and recent logs
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour) // 10 days ago
	recentTime := time.Now().UTC().Add(-1 * 24 * time.Hour) // 1 day ago

	oldLog := &domain.ActivityLog{
		Model:     domain.Model{ID: uuid.New().String(), CreatedAt: oldTime, UpdatedAt: oldTime},
		UserID:    uuid.New().String(),
		Action:    "old.action",
		Resource:  "test",
	}
	err = db.Create(oldLog).Error
	require.NoError(t, err)

	recentLog := &domain.ActivityLog{
		Model:     domain.Model{ID: uuid.New().String(), CreatedAt: recentTime, UpdatedAt: recentTime},
		UserID:    uuid.New().String(),
		Action:    "recent.action",
		Resource:  "test",
	}
	err = db.Create(recentLog).Error
	require.NoError(t, err)

	t.Run("delete logs older than 7 days", func(t *testing.T) {
		err := repo.DeleteOlderThan(ctx, 7)
		assert.NoError(t, err)

		// Check that old log was deleted but recent log remains
		var count int64
		db.Model(&domain.ActivityLog{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("delete logs older than 30 days", func(t *testing.T) {
		err := repo.DeleteOlderThan(ctx, 30)
		assert.NoError(t, err)

		// Both logs should remain
		var count int64
		db.Model(&domain.ActivityLog{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

// TestNewActivityRepository tests creating a new activity repository.
func TestNewActivityRepository(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewActivityRepository(db)
	assert.NotNil(t, repo)
}
