// Package repository_test provides tests for the auth user repository.
package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/repository"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Use a unique database for each test to avoid conflicts
	dbName := fmt.Sprintf("file:test_%s.db?mode=memory&cache=shared", uuid.NewV4().String())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Migrate the schema
	err = db.AutoMigrate(&domain.User{})
	require.NoError(t, err, "Failed to migrate test database")

	return db
}

// teardownTestDB closes the database connection.
func teardownTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")
	err = sqlDB.Close()
	require.NoError(t, err, "Failed to close test database")
}

// createTestUser creates a test user with a unique email.
func createTestUser(t *testing.T, db *gorm.DB) *domain.User {
	t.Helper()
	user := &domain.User{
		Model: domain.Model{
			ID:        uuid.NewV4().String(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Email:        fmt.Sprintf("user_%s@example.com", uuid.NewV4().String()),
		PasswordHash: "hashedpassword",
		Role:         domain.RoleUser,
		IsActive:     true,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful create user", func(t *testing.T) {
		user := &domain.User{
			Email:        fmt.Sprintf("test_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		// Verify user was created
		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("successful create admin user", func(t *testing.T) {
		user := &domain.User{
			Email:        fmt.Sprintf("admin_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleAdmin,
			IsActive:     true,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, user.Role)
	})

	t.Run("successful create inactive user", func(t *testing.T) {
		user := &domain.User{
			Email:        fmt.Sprintf("inactive_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     false,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.False(t, user.IsActive)
	})

	t.Run("successful create user with last login", func(t *testing.T) {
		now := time.Now().UTC()
		user := &domain.User{
			Email:        fmt.Sprintf("lastlogin_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
			LastLoginAt:  &now,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotNil(t, user.LastLoginAt)
	})
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful update user email", func(t *testing.T) {
		user := createTestUser(t, db)
		user.Email = fmt.Sprintf("updated_%s@example.com", uuid.NewV4().String())

		err := repo.Update(ctx, user)
		require.NoError(t, err)

		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("successful update user role", func(t *testing.T) {
		user := createTestUser(t, db)
		user.Role = domain.RoleAdmin

		err := repo.Update(ctx, user)
		require.NoError(t, err)

		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, found.Role)
	})

	t.Run("successful update user active status", func(t *testing.T) {
		user := createTestUser(t, db)
		user.IsActive = false

		err := repo.Update(ctx, user)
		require.NoError(t, err)

		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.False(t, found.IsActive)
	})

	t.Run("successful update password hash", func(t *testing.T) {
		user := createTestUser(t, db)
		user.PasswordHash = "newhash"

		err := repo.Update(ctx, user)
		require.NoError(t, err)

		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, "newhash", found.PasswordHash)
	})

	t.Run("successful update last login", func(t *testing.T) {
		user := createTestUser(t, db)
		now := time.Now().UTC()
		user.LastLoginAt = &now

		err := repo.Update(ctx, user)
		require.NoError(t, err)

		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.NotNil(t, found.LastLoginAt)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		user := &domain.User{
			Model: domain.Model{ID: uuid.NewV4().String()},
			Email: "nonexistent@example.com",
		}

		err := repo.Update(ctx, user)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestDelete tests the Delete (soft delete) method.
func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful soft delete user", func(t *testing.T) {
		user := createTestUser(t, db)

		err := repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// User should not be found in normal queries
		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// User should still exist in database (soft delete)
		var deleted domain.User
		err = db.Unscoped().Where("id = ?", user.ID).First(&deleted).Error
		require.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
		assert.True(t, deleted.DeletedAt.Valid)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("fail - already deleted user", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		err = repo.Delete(ctx, user.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestHardDelete tests the HardDelete method.
func TestHardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful hard delete user", func(t *testing.T) {
		user := createTestUser(t, db)

		err := repo.HardDelete(ctx, user.ID)
		require.NoError(t, err)

		// User should not exist at all (hard delete)
		var deleted domain.User
		err = db.Unscoped().Where("id = ?", user.ID).First(&deleted).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("successful hard delete soft-deleted user", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		err = repo.HardDelete(ctx, user.ID)
		require.NoError(t, err)

		// User should not exist at all
		var deleted domain.User
		err = db.Unscoped().Where("id = ?", user.ID).First(&deleted).Error
		assert.Error(t, err)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		err := repo.HardDelete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestRestore tests the Restore method.
func TestRestore(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful restore soft-deleted user", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		err = repo.Restore(ctx, user.ID)
		require.NoError(t, err)

		// User should now be findable in normal queries
		var found domain.User
		err = db.Where("id = ?", user.ID).First(&found).Error
		require.NoError(t, err)
		assert.False(t, found.DeletedAt.Valid)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		err := repo.Restore(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("fail - restore active user", func(t *testing.T) {
		user := createTestUser(t, db)

		err := repo.Restore(ctx, user.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestFindByID tests the FindByID method.
func TestFindByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful find active user by ID", func(t *testing.T) {
		user := createTestUser(t, db)

		found, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Role, found.Role)
	})

	t.Run("successful find admin user", func(t *testing.T) {
		user := createTestUser(t, db)
		user.Role = domain.RoleAdmin
		err := db.Save(user).Error
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, found.Role)
	})

	t.Run("successful find deleted user with include deleted", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID, &domain.ParanoidOptions{IncludeDeleted: true})
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("fail - find deleted user without include deleted", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.NewV4().String(), domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("successful find with nil options", func(t *testing.T) {
		user := createTestUser(t, db)

		found, err := repo.FindByID(ctx, user.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})
}

// TestFindByEmail tests the FindByEmail method.
func TestFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("successful find active user by email", func(t *testing.T) {
		user := createTestUser(t, db)

		found, err := repo.FindByEmail(ctx, user.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("successful find admin user by email", func(t *testing.T) {
		user := createTestUser(t, db)
		user.Role = domain.RoleAdmin
		err := db.Save(user).Error
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, user.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, found.Role)
	})

	t.Run("successful find inactive user", func(t *testing.T) {
		user := createTestUser(t, db)
		user.IsActive = false
		err := db.Save(user).Error
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, user.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.False(t, found.IsActive)
	})

	t.Run("successful find deleted user with include deleted", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, user.Email, &domain.ParanoidOptions{IncludeDeleted: true})
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("fail - find deleted user without include deleted", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		_, err = repo.FindByEmail(ctx, user.Email, domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("fail - user not found", func(t *testing.T) {
		_, err := repo.FindByEmail(ctx, "nonexistent@example.com", domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("successful find with nil options", func(t *testing.T) {
		user := createTestUser(t, db)

		found, err := repo.FindByEmail(ctx, user.Email, nil)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})
}

// TestFindAll tests the FindAll method.
func TestFindAll(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// Setup test data
	var users []*domain.User
	for i := 0; i < 5; i++ {
		user := createTestUser(t, db)
		users = append(users, user)
	}

	// Create a soft-deleted user
	deletedUser := createTestUser(t, db)
	err := db.Delete(deletedUser).Error
	require.NoError(t, err)

	t.Run("successful find all users - first page", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 2})
		require.NoError(t, err)
		assert.Len(t, result.Users, 2)
		assert.GreaterOrEqual(t, result.Total, int64(5))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("successful find all users - second page", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 2, Limit: 2})
		require.NoError(t, err)
		assert.Len(t, result.Users, 2)
		assert.Equal(t, 2, result.Page)
	})

	t.Run("successful find all users with role filter ADMIN", func(t *testing.T) {
		adminUser := createTestUser(t, db)
		adminUser.Role = domain.RoleAdmin
		err := db.Save(adminUser).Error
		require.NoError(t, err)

		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 10, Role: "ADMIN"})
		require.NoError(t, err)
		assert.Greater(t, result.Total, int64(0))
		for _, user := range result.Users {
			assert.Equal(t, domain.RoleAdmin, user.Role)
		}
	})

	t.Run("successful find all users with role filter USER", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 10, Role: "USER"})
		require.NoError(t, err)
		assert.Greater(t, result.Total, int64(0))
		for _, user := range result.Users {
			assert.Equal(t, domain.RoleUser, user.Role)
		}
	})

	t.Run("successful find all users with include deleted", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 10, IncludeDeleted: true})
		require.NoError(t, err)
		assert.Greater(t, result.Total, int64(5))
	})

	t.Run("successful find all users with only deleted", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 10, OnlyDeleted: true})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		if len(result.Users) > 0 {
			assert.NotNil(t, result.Users[0].DeletedAt)
			assert.True(t, result.Users[0].DeletedAt.Valid)
		}
	})

	t.Run("successful find all with nil request", func(t *testing.T) {
		result, err := repo.FindAll(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Total, int64(5))
	})

	t.Run("successful find all with pagination defaults", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 0, Limit: 0})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Limit)
	})

	t.Run("successful find all with empty result", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 1, Limit: 10, Search: "nonexistentuser123456"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Users)
	})
}

// TestExistsByEmail tests the ExistsByEmail method.
func TestExistsByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("email exists", func(t *testing.T) {
		user := createTestUser(t, db)

		exists, err := repo.ExistsByEmail(ctx, user.Email)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("email does not exist", func(t *testing.T) {
		exists, err := repo.ExistsByEmail(ctx, "nonexistent@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("soft deleted user should not exist", func(t *testing.T) {
		user := createTestUser(t, db)
		err := db.Delete(user).Error
		require.NoError(t, err)

		exists, err := repo.ExistsByEmail(ctx, user.Email)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("case sensitive email check", func(t *testing.T) {
		user := createTestUser(t, db)

		// Change case
		upperEmail := ""
		for i, c := range user.Email {
			if i == 0 {
				upperEmail += string(c - 32) // Convert to uppercase (ASCII only)
			} else {
				upperEmail += string(c)
			}
		}

		exists, err := repo.ExistsByEmail(ctx, upperEmail)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestAuthUserRepositoryIntegration tests the repository with multiple operations.
func TestAuthUserRepositoryIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("full lifecycle: create, find, update, delete, restore", func(t *testing.T) {
		// Create
		user := createTestUser(t, db)

		// Find by ID
		found, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)

		// Find by Email
		foundByEmail, err := repo.FindByEmail(ctx, user.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundByEmail.ID)

		// Update
		user.Role = domain.RoleAdmin
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, updated.Role)

		// Update last login
		now := time.Now().UTC()
		user.LastLoginAt = &now
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		// Verify last login
		withLogin, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.NotNil(t, withLogin.LastLoginAt)

		// Delete (soft)
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Verify soft delete
		_, err = repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)

		// Restore
		err = repo.Restore(ctx, user.ID)
		require.NoError(t, err)

		// Verify restore
		restored, err := repo.FindByID(ctx, user.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, restored.ID)

		// Hard delete
		err = repo.HardDelete(ctx, user.ID)
		require.NoError(t, err)

		// Verify hard delete
		_, err = repo.FindByID(ctx, user.ID, &domain.ParanoidOptions{IncludeDeleted: true})
		assert.Error(t, err)
	})

	t.Run("exists by email after operations", func(t *testing.T) {
		// Initial state - should not exist
		email := fmt.Sprintf("lifecycle_%s@example.com", uuid.NewV4().String())
		exists, err := repo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, exists)

		// Create user
		user := &domain.User{
			Email:        email,
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
		}
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		// Should exist now
		exists, err = repo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, exists)

		// Soft delete
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Should not exist after soft delete
		exists, err = repo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestEdgeCases tests edge cases and error conditions.
func TestEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("create user with empty ID", func(t *testing.T) {
		user := &domain.User{
			Email:        fmt.Sprintf("emptyid_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
	})

	t.Run("update non-existent user", func(t *testing.T) {
		user := &domain.User{
			Model: domain.Model{ID: uuid.NewV4().String()},
			Email: "nonexistent@example.com",
		}

		err := repo.Update(ctx, user)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("restore non-existent user", func(t *testing.T) {
		err := repo.Restore(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("hard delete non-existent user", func(t *testing.T) {
		err := repo.HardDelete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("FindAll with large page number", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{Page: 999, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Users)
	})
}

// TestUserRoleMethods tests user role-related methods.
func TestUserRoleMethods(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("create and find admin user", func(t *testing.T) {
		adminUser := &domain.User{
			Email:        fmt.Sprintf("admin_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleAdmin,
			IsActive:     true,
		}
		err := repo.Create(ctx, adminUser)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, adminUser.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.True(t, found.IsAdmin())
	})

	t.Run("create and find regular user", func(t *testing.T) {
		regularUser := &domain.User{
			Email:        fmt.Sprintf("user_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
		}
		err := repo.Create(ctx, regularUser)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, regularUser.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.False(t, found.IsAdmin())
	})
}

// TestUserActiveStatus tests user active status methods.
func TestUserActiveStatus(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("create and find active user", func(t *testing.T) {
		activeUser := &domain.User{
			Email:        fmt.Sprintf("active_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     true,
		}
		err := repo.Create(ctx, activeUser)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, activeUser.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.True(t, found.CanLogin())
	})

	t.Run("create and find inactive user", func(t *testing.T) {
		inactiveUser := &domain.User{
			Email:        fmt.Sprintf("inactive_%s@example.com", uuid.NewV4().String()),
			PasswordHash: "hashedpassword",
			Role:         domain.RoleUser,
			IsActive:     false,
		}
		err := repo.Create(ctx, inactiveUser)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, inactiveUser.Email, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.False(t, found.CanLogin())
	})
}
