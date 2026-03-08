// Package user provides integration tests for the user service.
package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/repository"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
)

// noopEventPublisher is a no-op implementation of eventbus.EventPublisher for integration tests.
type noopEventPublisher struct{}

func (n *noopEventPublisher) Publish(_ context.Context, _ string, _ *eventbus.Event) (string, error) {
	return "", nil
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t testing.TB) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Migrate tables
	err = db.AutoMigrate(&domain.User{}, &domain.Profile{}, &domain.ActivityLog{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	return db
}

// setupTestContext creates a test context with all dependencies.
func setupTestContext(t testing.TB) (*gorm.DB, usecase.UserUseCase, func()) {
	db := setupTestDB(t)

	// Create repositories
	userRepo := repository.NewUserRepository(db)
	activityRepo := repository.NewActivityRepository(db)

	// Create event bus (no-op for tests)
	eventPub := &noopEventPublisher{}

	// Create logger
	log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})

	// Create use case
	uc := usecase.NewUserUseCase(userRepo, activityRepo, eventPub, log)

	// Cleanup function
	cleanup := func() {
		db.Exec("DELETE FROM activity_logs")
		db.Exec("DELETE FROM profiles")
		db.Exec("DELETE FROM users")
	}

	return db, uc, cleanup
}

// TestUserRepository_Integration tests the user repository with real database.
func TestUserRepository_Integration(t *testing.T) {
	db, _, cleanup := setupTestContext(t)
	defer cleanup()

	repo := repository.NewUserRepository(db)

	t.Run("Create and Find User", func(t *testing.T) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Role:         domain.RoleUser,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		// Find by ID
		found, err := repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Role, found.Role)

		// Find by email
		foundByEmail, err := repo.FindByEmail(ctx, user.Email, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundByEmail.ID)
	})

	t.Run("Soft Delete and Restore", func(t *testing.T) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			Email:        "delete@example.com",
			Username:     "deleteuser",
			PasswordHash: "hashed_password",
			Role:         domain.RoleUser,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Soft delete
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Should not find without paranoid options
		_, err = repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		assert.Error(t, err)

		// Should find with include deleted
		found, err := repo.FindByID(ctx, user.ID, &dto.ParanoidOptions{IncludeDeleted: true})
		require.NoError(t, err)
		assert.NotNil(t, found)

		// Restore
		err = repo.Restore(ctx, user.ID)
		require.NoError(t, err)

		// Should find now
		found, err = repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("Update User", func(t *testing.T) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			Email:        "update@example.com",
			Username:     "updateuser",
			PasswordHash: "hashed_password",
			Role:         domain.RoleUser,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Update role
		user.Role = domain.RoleAdmin
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		// Verify
		found, err := repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, found.Role)
	})

	t.Run("Profile Management", func(t *testing.T) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			Email:        "profile@example.com",
			Username:     "profileuser",
			PasswordHash: "hashed_password",
			Role:         domain.RoleUser,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Create profile
		profile := &domain.Profile{
			UserID:    user.ID,
			FirstName: "John",
			LastName:  "Doe",
			Bio:       "Test user",
		}

		err = repo.UpdateProfile(ctx, profile)
		require.NoError(t, err)

		// Get profile
		found, err := repo.GetProfile(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "John", found.FirstName)
		assert.Equal(t, "Doe", found.LastName)

		// Update profile
		found.FirstName = "Jane"
		err = repo.UpdateProfile(ctx, found)
		require.NoError(t, err)

		// Verify
		updated, err := repo.GetProfile(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", updated.FirstName)
	})
}

// TestActivityRepository_Integration tests the activity repository.
func TestActivityRepository_Integration(t *testing.T) {
	db, _, cleanup := setupTestContext(t)
	defer cleanup()

	repo := repository.NewActivityRepository(db)

	t.Run("Create and Find Activity", func(t *testing.T) {
		ctx := context.Background()

		// Create activity logs
		activity1 := domain.NewActivityLog("user-1", "login", "auth", "session-1").
			WithRequestInfo("127.0.0.1", "test-agent")

		activity2 := domain.NewActivityLog("user-1", "logout", "auth", "session-1").
			WithRequestInfo("127.0.0.1", "test-agent")

		err := repo.Create(ctx, activity1)
		require.NoError(t, err)

		err = repo.Create(ctx, activity2)
		require.NoError(t, err)

		// Find by user ID
		req := &dto.ListActivityLogsRequest{
			UserID: "user-1",
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindByUserID(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Logs, 2)
	})

	t.Run("Filter by Action", func(t *testing.T) {
		ctx := context.Background()

		// Create activity logs
		activity1 := domain.NewActivityLog("user-2", "login", "auth", "")
		activity2 := domain.NewActivityLog("user-2", "profile_update", "profile", "")

		err := repo.Create(ctx, activity1)
		require.NoError(t, err)

		err = repo.Create(ctx, activity2)
		require.NoError(t, err)

		// Find by user ID and action
		req := &dto.ListActivityLogsRequest{
			UserID: "user-2",
			Action: "login",
			Page:   1,
			Limit:  10,
		}

		result, err := repo.FindByUserID(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Len(t, result.Logs, 1)
		assert.Equal(t, "login", result.Logs[0].Action)
	})
}

// TestUserUseCase_Integration tests the user use case with real dependencies.
func TestUserUseCase_Integration(t *testing.T) {
	_, uc, cleanup := setupTestContext(t)
	defer cleanup()

	t.Run("Complete User Lifecycle", func(t *testing.T) {
		ctx := context.Background()

		// This is a simplified integration test
		// In a real scenario, you'd test the full flow including:
		// - Create user
		// - Update profile
		// - Activate/deactivate
		// - Soft delete
		// - Restore

		// For now, we'll test that the use case is properly wired
		assert.NotNil(t, uc)
		_ = ctx // suppress unused variable
	})
}

// TestConcurrentOperations tests concurrent database operations.
func TestConcurrentOperations(t *testing.T) {
	db, _, cleanup := setupTestContext(t)
	defer cleanup()

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("Concurrent User Creation", func(t *testing.T) {
		// Create multiple users concurrently
		numUsers := 10
		errChan := make(chan error, numUsers)

		for i := 0; i < numUsers; i++ {
			go func(index int) {
				user := &domain.User{
					Email:        fmt.Sprintf("concurrent%d@example.com", index),
					Username:     fmt.Sprintf("concurrentuser%d", index),
					PasswordHash: "hashed_password",
					Role:         domain.RoleUser,
				}
				errChan <- repo.Create(ctx, user)
			}(i)
		}

		// Collect errors
		for i := 0; i < numUsers; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// Verify all users were created
		req := &dto.ListUsersRequest{
			Page:  1,
			Limit: 100,
		}

		result, err := repo.FindAll(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(numUsers), result.Total)
	})
}

// BenchmarkRepositoryOperations benchmarks repository operations.
func BenchmarkRepositoryOperations(b *testing.B) {
	db, _, cleanup := setupTestContext(b)
	defer cleanup()

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	b.Run("CreateUser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := &domain.User{
				Email:        fmt.Sprintf("bench%d@example.com", i),
				PasswordHash: "hashed_password",
				Role:         domain.RoleUser,
			}
			_ = repo.Create(ctx, user)
		}
	})

	b.Run("FindUser", func(b *testing.B) {
		// Create a user first
		user := &domain.User{
			Email:        "find@example.com",
			PasswordHash: "hashed_password",
			Role:         domain.RoleUser,
		}
		_ = repo.Create(ctx, user)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		}
	})
}
