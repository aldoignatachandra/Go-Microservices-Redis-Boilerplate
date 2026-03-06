// Package repository_test provides tests for the user repository.
package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/repository"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Migrate the schema
	err = db.AutoMigrate(&domain.User{}, &domain.Profile{})
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

// UserBuilder is a builder pattern for creating test users.
type UserBuilder struct {
	user *domain.User
}

// NewUserBuilder creates a new UserBuilder.
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		user: &domain.User{
			Model: domain.Model{
				ID:        "test-user-id",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			Email:    "test@example.com",
			Role:     domain.RoleUser,
			IsActive: true,
		},
	}
}

// WithID sets the user ID.
func (b *UserBuilder) WithID(id string) *UserBuilder {
	b.user.ID = id
	return b
}

// WithEmail sets the user email.
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

// WithRole sets the user role.
func (b *UserBuilder) WithRole(role domain.Role) *UserBuilder {
	b.user.Role = role
	return b
}

// WithIsActive sets the user active status.
func (b *UserBuilder) WithIsActive(isActive bool) *UserBuilder {
	b.user.IsActive = isActive
	return b
}

// WithPasswordHash sets the password hash.
func (b *UserBuilder) WithPasswordHash(hash string) *UserBuilder {
	b.user.PasswordHash = hash
	return b
}

// WithProfile adds a profile to the user.
func (b *UserBuilder) WithProfile(profile *domain.Profile) *UserBuilder {
	b.user.Profile = profile
	return b
}

// Build returns the constructed user.
func (b *UserBuilder) Build() *domain.User {
	return b.user
}

// ProfileBuilder is a builder pattern for creating test profiles.
type ProfileBuilder struct {
	profile *domain.Profile
}

// NewProfileBuilder creates a new ProfileBuilder.
func NewProfileBuilder() *ProfileBuilder {
	return &ProfileBuilder{
		profile: &domain.Profile{
			Model: domain.Model{
				ID:        "test-profile-id",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			UserID:    "test-user-id",
			FirstName: "John",
			LastName:  "Doe",
		},
	}
}

// WithID sets the profile ID.
func (b *ProfileBuilder) WithID(id string) *ProfileBuilder {
	b.profile.ID = id
	return b
}

// WithUserID sets the user ID.
func (b *ProfileBuilder) WithUserID(userID string) *ProfileBuilder {
	b.profile.UserID = userID
	return b
}

// WithFirstName sets the first name.
func (b *ProfileBuilder) WithFirstName(firstName string) *ProfileBuilder {
	b.profile.FirstName = firstName
	return b
}

// WithLastName sets the last name.
func (b *ProfileBuilder) WithLastName(lastName string) *ProfileBuilder {
	b.profile.LastName = lastName
	return b
}

// WithBio sets the bio.
func (b *ProfileBuilder) WithBio(bio string) *ProfileBuilder {
	b.profile.Bio = bio
	return b
}

// WithAvatar sets the avatar.
func (b *ProfileBuilder) WithAvatar(avatar string) *ProfileBuilder {
	b.profile.Avatar = avatar
	return b
}

// Build returns the constructed profile.
func (b *ProfileBuilder) Build() *domain.Profile {
	return b.profile
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		user        *domain.User
		setup       func(*gorm.DB)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful create user",
			user: NewUserBuilder().
				WithEmail("user1@example.com").
				WithPasswordHash("hashedpassword").
				Build(),
			setup:   func(db *gorm.DB) {},
			wantErr: false,
		},
		{
			name: "successful create user with profile",
			user: NewUserBuilder().
				WithID("user-with-profile").
				WithEmail("user2@example.com").
				WithPasswordHash("hashedpassword").
				WithProfile(NewProfileBuilder().
					WithUserID("user-with-profile").
					WithFirstName("Jane").
					WithLastName("Smith").
					Build()).
				Build(),
			setup:   func(db *gorm.DB) {},
			wantErr: false,
		},
		{
			name: "successful create admin user",
			user: NewUserBuilder().
				WithID("admin-user").
				WithEmail("admin@example.com").
				WithRole(domain.RoleAdmin).
				WithPasswordHash("hashedpassword").
				Build(),
			setup:   func(db *gorm.DB) {},
			wantErr: false,
		},
		{
			name: "successful create inactive user",
			user: NewUserBuilder().
				WithID("inactive-user").
				WithEmail("inactive@example.com").
				WithIsActive(false).
				WithPasswordHash("hashedpassword").
				Build(),
			setup:   func(db *gorm.DB) {},
			wantErr: false,
		},
		{
			name: "fail - duplicate email",
			user: NewUserBuilder().
				WithID("duplicate-user").
				WithEmail("user1@example.com").
				WithPasswordHash("hashedpassword").
				Build(),
			setup: func(db *gorm.DB) {
				// Create a user with the same email first
				existingUser := NewUserBuilder().
					WithEmail("user1@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(existingUser).Error
				require.NoError(t, err)
			},
			wantErr:     true,
			expectedErr: domain.ErrEmailAlreadyUsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tt.setup(db)

			// Act
			err := repo.Create(ctx, tt.user)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, tt.user.ID, "User ID should be set")

				// Verify user was actually created in the database
				var found domain.User
				err = db.Where("id = ?", tt.user.ID).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, tt.user.Email, found.Email)
				assert.Equal(t, tt.user.Role, found.Role)
				assert.Equal(t, tt.user.IsActive, found.IsActive)
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() *domain.User
		updateUser  *domain.User
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *gorm.DB, string)
	}{
		{
			name: "successful update user email",
			setup: func() *domain.User {
				user := NewUserBuilder().
					WithEmail("old@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user
			},
			updateUser: func() *domain.User {
				user := NewUserBuilder().
					WithEmail("new@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				// Set the ID from the setup
				var existing domain.User
				err := db.Where("email = ?", "old@example.com").First(&existing).Error
				require.NoError(t, err)
				user.ID = existing.ID
				return user
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, "new@example.com", found.Email)
			},
		},
		{
			name: "successful update user role",
			setup: func() *domain.User {
				user := NewUserBuilder().
					WithEmail("roleuser@example.com").
					WithRole(domain.RoleUser).
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user
			},
			updateUser: func() *domain.User {
				var existing domain.User
				err := db.Where("email = ?", "roleuser@example.com").First(&existing).Error
				require.NoError(t, err)
				existing.Role = domain.RoleAdmin
				return &existing
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, domain.RoleAdmin, found.Role)
			},
		},
		{
			name: "successful update user active status",
			setup: func() *domain.User {
				user := NewUserBuilder().
					WithEmail("activeuser@example.com").
					WithIsActive(true).
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user
			},
			updateUser: func() *domain.User {
				var existing domain.User
				err := db.Where("email = ?", "activeuser@example.com").First(&existing).Error
				require.NoError(t, err)
				existing.IsActive = false
				return &existing
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				require.NoError(t, err)
				assert.False(t, found.IsActive)
			},
		},
		{
			name: "fail - user not found",
			setup: func() *domain.User {
				return nil
			},
			updateUser: NewUserBuilder().
				WithID("non-existent-id").
				WithEmail("nonexistent@example.com").
				WithPasswordHash("hashedpassword").
				Build(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var userID string
			if tt.setup != nil {
				user := tt.setup()
				if user != nil {
					userID = user.ID
				}
			}
			if tt.updateUser != nil {
				userID = tt.updateUser.ID
			}

			// Act
			err := repo.Update(ctx, tt.updateUser)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, db, userID)
				}
			}
		})
	}
}

// TestDelete tests the Delete (soft delete) method.
func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		deleteID    string
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *gorm.DB, string)
	}{
		{
			name: "successful soft delete user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("todelete@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			deleteID: func() string {
				var user domain.User
				err := db.Where("email = ?", "todelete@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				// User should not be found in normal queries
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				assert.Error(t, err)
				assert.Equal(t, gorm.ErrRecordNotFound, err)

				// User should still exist in database (soft delete)
				var deleted domain.User
				err = db.Unscoped().Where("id = ?", id).First(&deleted).Error
				require.NoError(t, err)
				assert.NotNil(t, deleted.DeletedAt)
				assert.True(t, deleted.DeletedAt.Valid)
			},
		},
		{
			name: "fail - user not found",
			setup: func() string {
				return ""
			},
			deleteID:    "non-existent-id",
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name: "fail - already deleted user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("alreadydeleted@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete the user
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.ID
			},
			deleteID: func() string {
				var user domain.User
				err := db.Unscoped().Where("email = ?", "alreadydeleted@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			err := repo.Delete(ctx, tt.deleteID)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, db, tt.deleteID)
				}
			}
		})
	}
}

// TestHardDelete tests the HardDelete method.
func TestHardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		deleteID    string
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *gorm.DB, string)
	}{
		{
			name: "successful hard delete user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("toharddelete@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			deleteID: func() string {
				var user domain.User
				err := db.Where("email = ?", "toharddelete@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				// User should not be found in normal queries
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				assert.Error(t, err)

				// User should not exist at all (hard delete)
				var deleted domain.User
				err = db.Unscoped().Where("id = ?", id).First(&deleted).Error
				assert.Error(t, err)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			},
		},
		{
			name: "successful hard delete soft-deleted user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("softdeleted@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete first
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.ID
			},
			deleteID: func() string {
				var user domain.User
				err := db.Unscoped().Where("email = ?", "softdeleted@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				// User should not exist at all (hard delete)
				var deleted domain.User
				err := db.Unscoped().Where("id = ?", id).First(&deleted).Error
				assert.Error(t, err)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			},
		},
		{
			name: "fail - user not found",
			setup: func() string {
				return ""
			},
			deleteID:    "non-existent-id",
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			err := repo.HardDelete(ctx, tt.deleteID)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, db, tt.deleteID)
				}
			}
		})
	}
}

// TestRestore tests the Restore method.
func TestRestore(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		restoreID   string
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *gorm.DB, string)
	}{
		{
			name: "successful restore soft-deleted user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("torestore@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete the user
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.ID
			},
			restoreID: func() string {
				var user domain.User
				err := db.Unscoped().Where("email = ?", "torestore@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, id string) {
				// User should now be findable in normal queries
				var found domain.User
				err := db.Where("id = ?", id).First(&found).Error
				require.NoError(t, err)
				assert.False(t, found.DeletedAt.Valid)
			},
		},
		{
			name: "fail - user not found",
			setup: func() string {
				return ""
			},
			restoreID:   "non-existent-id",
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name: "fail - restore active user",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("active@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			restoreID: func() string {
				var user domain.User
				err := db.Where("email = ?", "active@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			err := repo.Restore(ctx, tt.restoreID)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, db, tt.restoreID)
				}
			}
		})
	}
}

// TestFindByID tests the FindByID method.
func TestFindByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		findID      string
		opts        *dto.ParanoidOptions
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *domain.User)
	}{
		{
			name: "successful find active user by ID",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("findbyid@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			findID: func() string {
				var user domain.User
				err := db.Where("email = ?", "findbyid@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			opts:    dto.DefaultParanoidOptions(),
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "findbyid@example.com", user.Email)
				assert.Equal(t, domain.RoleUser, user.Role)
				assert.True(t, user.IsActive)
			},
		},
		{
			name: "successful find user with profile",
			setup: func() string {
				userID := "user-with-profile-id"
				profile := NewProfileBuilder().
					WithUserID(userID).
					WithFirstName("John").
					WithLastName("Doe").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)

				user := NewUserBuilder().
					WithID(userID).
					WithEmail("withprofile@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err = db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			findID: func() string {
				var user domain.User
				err := db.Where("email = ?", "withprofile@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			opts:    dto.DefaultParanoidOptions(),
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.NotNil(t, user.Profile)
				assert.Equal(t, "John", user.Profile.FirstName)
				assert.Equal(t, "Doe", user.Profile.LastName)
			},
		},
		{
			name: "successful find deleted user with include deleted",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("deleteduser@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.ID
			},
			findID: func() string {
				var user domain.User
				err := db.Unscoped().Where("email = ?", "deleteduser@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			opts: &dto.ParanoidOptions{
				IncludeDeleted: true,
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "deleteduser@example.com", user.Email)
			},
		},
		{
			name: "fail - find deleted user without include deleted",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("deleteduser2@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.ID
			},
			findID: func() string {
				var user domain.User
				err := db.Unscoped().Where("email = ?", "deleteduser2@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			opts:        dto.DefaultParanoidOptions(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name:        "fail - user not found",
			setup:       func() string { return "" },
			findID:      "non-existent-id",
			opts:        dto.DefaultParanoidOptions(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name: "successful find with nil options (uses defaults)",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("nilopts@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.ID
			},
			findID: func() string {
				var user domain.User
				err := db.Where("email = ?", "nilopts@example.com").First(&user).Error
				require.NoError(t, err)
				return user.ID
			}(),
			opts:    nil,
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "nilopts@example.com", user.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			user, err := repo.FindByID(ctx, tt.findID, tt.opts)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.findID, user.ID)
				if tt.validate != nil {
					tt.validate(t, user)
				}
			}
		})
	}
}

// TestFindByEmail tests the FindByEmail method.
func TestFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		findEmail   string
		opts        *dto.ParanoidOptions
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *domain.User)
	}{
		{
			name: "successful find active user by email",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("findbyemail@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.Email
			},
			findEmail: "findbyemail@example.com",
			opts:      dto.DefaultParanoidOptions(),
			wantErr:   false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "findbyemail@example.com", user.Email)
				assert.Equal(t, domain.RoleUser, user.Role)
			},
		},
		{
			name: "successful find user with profile by email",
			setup: func() string {
				userID := "user-with-profile-2-id"
				profile := NewProfileBuilder().
					WithUserID(userID).
					WithFirstName("Jane").
					WithLastName("Smith").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)

				user := NewUserBuilder().
					WithID(userID).
					WithEmail("withprofile2@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err = db.Create(user).Error
				require.NoError(t, err)
				return user.Email
			},
			findEmail: "withprofile2@example.com",
			opts:      dto.DefaultParanoidOptions(),
			wantErr:   false,
			validate: func(t *testing.T, user *domain.User) {
				assert.NotNil(t, user.Profile)
				assert.Equal(t, "Jane", user.Profile.FirstName)
				assert.Equal(t, "Smith", user.Profile.LastName)
			},
		},
		{
			name: "successful find deleted user with include deleted",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("deletedemail@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.Email
			},
			findEmail: "deletedemail@example.com",
			opts: &dto.ParanoidOptions{
				IncludeDeleted: true,
			},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "deletedemail@example.com", user.Email)
			},
		},
		{
			name: "fail - find deleted user without include deleted",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("deletedemail2@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				// Soft delete
				err = db.Delete(user).Error
				require.NoError(t, err)
				return user.Email
			},
			findEmail:   "deletedemail2@example.com",
			opts:        dto.DefaultParanoidOptions(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name:        "fail - user not found",
			setup:       func() string { return "" },
			findEmail:   "nonexistent@example.com",
			opts:        dto.DefaultParanoidOptions(),
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name: "successful find with nil options (uses defaults)",
			setup: func() string {
				user := NewUserBuilder().
					WithEmail("niloptsemail@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				return user.Email
			},
			findEmail: "niloptsemail@example.com",
			opts:      nil,
			wantErr:   false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "niloptsemail@example.com", user.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			user, err := repo.FindByEmail(ctx, tt.findEmail, tt.opts)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.findEmail, user.Email)
				if tt.validate != nil {
					tt.validate(t, user)
				}
			}
		})
	}
}

// TestFindAll tests the FindAll method.
func TestFindAll(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// Setup test data
	setupTestData := func() {
		users := []*domain.User{
			NewUserBuilder().
				WithID("user1").
				WithEmail("user1@example.com").
				WithRole(domain.RoleUser).
				WithPasswordHash("hash1").
				Build(),
			NewUserBuilder().
				WithID("user2").
				WithEmail("user2@example.com").
				WithRole(domain.RoleAdmin).
				WithPasswordHash("hash2").
				Build(),
			NewUserBuilder().
				WithID("user3").
				WithEmail("user3@example.com").
				WithRole(domain.RoleUser).
				WithPasswordHash("hash3").
				Build(),
			NewUserBuilder().
				WithID("user4").
				WithEmail("john@example.com").
				WithRole(domain.RoleUser).
				WithPasswordHash("hash4").
				Build(),
		}
		for _, user := range users {
			err := db.Create(user).Error
			require.NoError(t, err)
		}

		// Add profiles for search testing
		profile := NewProfileBuilder().
			WithUserID("user4").
			WithFirstName("John").
			WithLastName("Doe").
			Build()
		err := db.Create(profile).Error
		require.NoError(t, err)

		// Create a soft-deleted user
		deletedUser := NewUserBuilder().
			WithID("deleted1").
			WithEmail("deleted@example.com").
			WithRole(domain.RoleUser).
			WithPasswordHash("hash_deleted").
			Build()
		err = db.Create(deletedUser).Error
		require.NoError(t, err)
		err = db.Delete(deletedUser).Error
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		setup       func()
		req         *dto.ListUsersRequest
		wantErr     bool
		validate    func(*testing.T, *domain.UserList)
		checkTotal  int
		checkPage   int
		checkLimit  int
	}{
		{
			name:  "successful find all users - first page",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  1,
				Limit: 2,
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Len(t, result.Users, 2)
				assert.GreaterOrEqual(t, result.Total, int64(4))
			},
			checkPage:  1,
			checkLimit: 2,
		},
		{
			name:  "successful find all users - second page",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  2,
				Limit: 2,
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Len(t, result.Users, 2)
				assert.Equal(t, 2, result.Page)
			},
			checkPage:  2,
			checkLimit: 2,
		},
		{
			name:  "successful find all users with role filter",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  1,
				Limit: 10,
				Role:  "ADMIN",
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Greater(t, result.Total, int64(0))
				for _, user := range result.Users {
					assert.Equal(t, domain.RoleAdmin, user.Role)
				}
			},
		},
		{
			name:  "successful find all users with role filter USER",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  1,
				Limit: 10,
				Role:  "USER",
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Greater(t, result.Total, int64(0))
				for _, user := range result.Users {
					assert.Equal(t, domain.RoleUser, user.Role)
				}
			},
		},
		{
			name:  "successful find all users with email search",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  1,
				Limit: 10,
				Search: "user1",
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.GreaterOrEqual(t, result.Total, int64(1))
				assert.Contains(t, result.Users[0].Email, "user1")
			},
		},
		{
			name:  "successful find all users with name search",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  1,
				Limit: 10,
				Search: "John",
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.GreaterOrEqual(t, result.Total, int64(1))
			},
		},
		{
			name:  "successful find all users with include deleted",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:           1,
				Limit:          10,
				IncludeDeleted: true,
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Greater(t, result.Total, int64(4))
			},
		},
		{
			name:  "successful find all users with only deleted",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:        1,
				Limit:       10,
				OnlyDeleted: true,
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.Equal(t, int64(1), result.Total)
				if len(result.Users) > 0 {
					assert.NotNil(t, result.Users[0].DeletedAt)
					assert.True(t, result.Users[0].DeletedAt.Valid)
				}
			},
		},
		{
			name:  "successful find all with nil request",
			setup: setupTestData,
			req:   nil,
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.NotNil(t, result)
				assert.GreaterOrEqual(t, result.Total, int64(4))
			},
		},
		{
			name:  "successful find all with pagination defaults",
			setup: setupTestData,
			req: &dto.ListUsersRequest{
				Page:  0, // Should default to 1
				Limit: 0, // Should default to 10
			},
			wantErr: false,
			validate: func(t *testing.T, result *domain.UserList) {
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.Page)
				assert.Equal(t, 10, result.Limit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			result, err := repo.FindAll(ctx, tt.req)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.checkPage, result.Page)
				assert.Equal(t, tt.checkLimit, result.Limit)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// TestExistsByEmail tests the ExistsByEmail method.
func TestExistsByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func()
		email      string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "email exists",
			setup: func() {
				user := NewUserBuilder().
					WithEmail("existing@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
			},
			email:      "existing@example.com",
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "email does not exist",
			setup:      func() {},
			email:      "nonexistent@example.com",
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "soft deleted user should not exist",
			setup: func() {
				user := NewUserBuilder().
					WithEmail("softdeleted@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
				err = db.Delete(user).Error
				require.NoError(t, err)
			},
			email:      "softdeleted@example.com",
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "case sensitive email check",
			setup: func() {
				user := NewUserBuilder().
					WithEmail("Test@example.com").
					WithPasswordHash("hashedpassword").
					Build()
				err := db.Create(user).Error
				require.NoError(t, err)
			},
			email:      "test@example.com", // Different case
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tt.setup()

			// Act
			exists, err := repo.ExistsByEmail(ctx, tt.email)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
		})
	}
}

// TestUpdateProfile tests the UpdateProfile method.
func TestUpdateProfile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() *domain.Profile
		profile     *domain.Profile
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *gorm.DB, string)
	}{
		{
			name: "successful update existing profile",
			setup: func() *domain.Profile {
				profile := NewProfileBuilder().
					WithUserID("user-up-profile").
					WithFirstName("John").
					WithLastName("Doe").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)
				return profile
			},
			profile: func() *domain.Profile {
				var profile domain.Profile
				err := db.Where("user_id = ?", "user-up-profile").First(&profile).Error
				require.NoError(t, err)
				profile.FirstName = "Jane"
				profile.LastName = "Smith"
				return &profile
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, userID string) {
				var found domain.Profile
				err := db.Where("user_id = ?", userID).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, "Jane", found.FirstName)
				assert.Equal(t, "Smith", found.LastName)
			},
		},
		{
			name:  "successful create new profile",
			setup: func() *domain.Profile { return nil },
			profile: NewProfileBuilder().
				WithID("new-profile-id").
				WithUserID("user-new-profile").
				WithFirstName("Alice").
				WithLastName("Johnson").
				Build(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, userID string) {
				var found domain.Profile
				err := db.Where("user_id = ?", userID).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, "Alice", found.FirstName)
				assert.Equal(t, "Johnson", found.LastName)
			},
		},
		{
			name: "successful update profile with bio",
			setup: func() *domain.Profile {
				profile := NewProfileBuilder().
					WithUserID("user-bio").
					WithFirstName("Bob").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)
				return profile
			},
			profile: func() *domain.Profile {
				var profile domain.Profile
				err := db.Where("user_id = ?", "user-bio").First(&profile).Error
				require.NoError(t, err)
				profile.Bio = "Software Engineer"
				return &profile
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, userID string) {
				var found domain.Profile
				err := db.Where("user_id = ?", userID).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, "Software Engineer", found.Bio)
			},
		},
		{
			name: "successful update profile with avatar",
			setup: func() *domain.Profile {
				profile := NewProfileBuilder().
					WithUserID("user-avatar").
					WithFirstName("Charlie").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)
				return profile
			},
			profile: func() *domain.Profile {
				var profile domain.Profile
				err := db.Where("user_id = ?", "user-avatar").First(&profile).Error
				require.NoError(t, err)
				profile.Avatar = "https://example.com/avatar.jpg"
				return &profile
			}(),
			wantErr: false,
			validate: func(t *testing.T, db *gorm.DB, userID string) {
				var found domain.Profile
				err := db.Where("user_id = ?", userID).First(&found).Error
				require.NoError(t, err)
				assert.Equal(t, "https://example.com/avatar.jpg", found.Avatar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var userID string
			if tt.setup != nil {
				profile := tt.setup()
				if profile != nil {
					userID = profile.UserID
				}
			}
			if tt.profile != nil {
				userID = tt.profile.UserID
			}

			// Act
			err := repo.UpdateProfile(ctx, tt.profile)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, db, userID)
				}
			}
		})
	}
}

// TestGetProfile tests the GetProfile method.
func TestGetProfile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() string
		userID      string
		wantErr     bool
		expectedErr error
		validate    func(*testing.T, *domain.Profile)
	}{
		{
			name: "successful get existing profile",
			setup: func() string {
				profile := NewProfileBuilder().
					WithUserID("user-get-profile").
					WithFirstName("John").
					WithLastName("Doe").
					WithBio("Test user").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)
				return profile.UserID
			},
			userID:  "user-get-profile",
			wantErr: false,
			validate: func(t *testing.T, profile *domain.Profile) {
				require.NotNil(t, profile)
				assert.Equal(t, "user-get-profile", profile.UserID)
				assert.Equal(t, "John", profile.FirstName)
				assert.Equal(t, "Doe", profile.LastName)
				assert.Equal(t, "Test user", profile.Bio)
			},
		},
		{
			name:        "successful get non-existent profile returns nil",
			setup:       func() string { return "" },
			userID:      "non-existent-user",
			wantErr:     false,
			validate:    func(t *testing.T, profile *domain.Profile) { assert.Nil(t, profile) },
		},
		{
			name: "successful get profile with all fields",
			setup: func() string {
				profile := NewProfileBuilder().
					WithUserID("user-full-profile").
					WithFirstName("Jane").
					WithLastName("Smith").
					WithBio("Full stack developer").
					WithAvatar("https://example.com/jane.jpg").
					Build()
				err := db.Create(profile).Error
				require.NoError(t, err)
				return profile.UserID
			},
			userID:  "user-full-profile",
			wantErr: false,
			validate: func(t *testing.T, profile *domain.Profile) {
				require.NotNil(t, profile)
				assert.Equal(t, "user-full-profile", profile.UserID)
				assert.Equal(t, "Jane", profile.FirstName)
				assert.Equal(t, "Smith", profile.LastName)
				assert.Equal(t, "Full stack developer", profile.Bio)
				assert.Equal(t, "https://example.com/jane.jpg", profile.Avatar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.setup != nil {
				tt.setup()
			}

			// Act
			profile, err := repo.GetProfile(ctx, tt.userID)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, profile)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, profile)
				}
			}
		})
	}
}

// TestUserRepositoryIntegration tests the repository with multiple operations.
func TestUserRepositoryIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("full lifecycle: create, find, update, delete, restore", func(t *testing.T) {
		// Create
		user := NewUserBuilder().
			WithEmail("lifecycle@example.com").
			WithPasswordHash("hashedpassword").
			Build()
		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		// Find by ID
		found, err := repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)

		// Find by Email
		foundByEmail, err := repo.FindByEmail(ctx, user.Email, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundByEmail.ID)

		// Update
		user.Role = domain.RoleAdmin
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, updated.Role)

		// Create and update profile
		profile := NewProfileBuilder().
			WithUserID(user.ID).
			WithFirstName("Test").
			WithLastName("User").
			Build()
		err = repo.UpdateProfile(ctx, profile)
		require.NoError(t, err)

		// Get profile
		retrievedProfile, err := repo.GetProfile(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedProfile)
		assert.Equal(t, "Test", retrievedProfile.FirstName)

		// Delete (soft)
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Verify soft delete
		_, err = repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)

		// Restore
		err = repo.Restore(ctx, user.ID)
		require.NoError(t, err)

		// Verify restore
		restored, err := repo.FindByID(ctx, user.ID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, user.ID, restored.ID)

		// Hard delete
		err = repo.HardDelete(ctx, user.ID)
		require.NoError(t, err)

		// Verify hard delete
		_, err = repo.FindByID(ctx, user.ID, &dto.ParanoidOptions{IncludeDeleted: true})
		assert.Error(t, err)
	})

	t.Run("exists by email after operations", func(t *testing.T) {
		// Initial state - should not exist
		exists, err := repo.ExistsByEmail(ctx, "existscheck@example.com")
		require.NoError(t, err)
		assert.False(t, exists)

		// Create user
		user := NewUserBuilder().
			WithEmail("existscheck@example.com").
			WithPasswordHash("hashedpassword").
			Build()
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		// Should exist now
		exists, err = repo.ExistsByEmail(ctx, "existscheck@example.com")
		require.NoError(t, err)
		assert.True(t, exists)

		// Soft delete
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Should not exist after soft delete
		exists, err = repo.ExistsByEmail(ctx, "existscheck@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// strPtr is a helper function to create a string pointer.
func strPtr(s string) *string {
	return &s
}

// TestUserRepositoryWithProfilePreloading tests profile preloading behavior.
func TestUserRepositoryWithProfilePreloading(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("FindByID preloads profile", func(t *testing.T) {
		// Setup: Create user with profile
		userID := "preload-test-user"
		profile := NewProfileBuilder().
			WithUserID(userID).
			WithFirstName("Preload").
			WithLastName("Test").
			Build()
		err := db.Create(profile).Error
		require.NoError(t, err)

		user := NewUserBuilder().
			WithID(userID).
			WithEmail("preload@example.com").
			WithPasswordHash("hashedpassword").
			Build()
		err = db.Create(user).Error
		require.NoError(t, err)

		// Find user
		found, err := repo.FindByID(ctx, userID, dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.NotNil(t, found.Profile)
		assert.Equal(t, "Preload", found.Profile.FirstName)
	})

	t.Run("FindByEmail preloads profile", func(t *testing.T) {
		// Setup: Create user with profile
		userID := "preload-email-test-user"
		profile := NewProfileBuilder().
			WithUserID(userID).
			WithFirstName("Email").
			WithLastName("Preload").
			Build()
		err := db.Create(profile).Error
		require.NoError(t, err)

		user := NewUserBuilder().
			WithID(userID).
			WithEmail("emailpreload@example.com").
			WithPasswordHash("hashedpassword").
			Build()
		err = db.Create(user).Error
		require.NoError(t, err)

		// Find user by email
		found, err := repo.FindByEmail(ctx, "emailpreload@example.com", dto.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.NotNil(t, found.Profile)
		assert.Equal(t, "Email", found.Profile.FirstName)
	})

	t.Run("FindAll preloads profiles", func(t *testing.T) {
		// Setup: Create multiple users with profiles
		for i := 1; i <= 3; i++ {
			userID := "preload-all-" + string(rune('0'+i))
			profile := NewProfileBuilder().
				WithUserID(userID).
				WithFirstName("User" + string(rune('0'+i))).
				Build()
			err := db.Create(profile).Error
			require.NoError(t, err)

			user := NewUserBuilder().
				WithID(userID).
				WithEmail("preloadall" + string(rune('0'+i)) + "@example.com").
				WithPasswordHash("hashedpassword").
				Build()
			err = db.Create(user).Error
			require.NoError(t, err)
		}

		// Find all
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{
			Page:  1,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Users), 3)
	})
}

// TestEdgeCases tests edge cases and error conditions.
func TestEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("create user with empty ID (should generate)", func(t *testing.T) {
		user := NewUserBuilder().
			WithID("").
			WithEmail("emptyid@example.com").
			WithPasswordHash("hashedpassword").
			Build()

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID, "ID should be generated by BeforeCreate hook")
	})

	t.Run("update non-existent user", func(t *testing.T) {
		user := NewUserBuilder().
			WithID("totally-non-existent-id").
			WithEmail("nonexistent@example.com").
			WithPasswordHash("hashedpassword").
			Build()

		err := repo.Update(ctx, user)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		err := repo.Delete(ctx, "totally-non-existent-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("restore non-existent user", func(t *testing.T) {
		err := repo.Restore(ctx, "totally-non-existent-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("hard delete non-existent user", func(t *testing.T) {
		err := repo.HardDelete(ctx, "totally-non-existent-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("FindAll with empty result", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{
			Page:  1,
			Limit: 10,
			Role:  "ADMIN",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Users)
	})

	t.Run("FindAll with large page number", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListUsersRequest{
			Page:  999,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Users)
	})
}

// TestContextCancellation tests behavior with cancelled context.
func TestContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)

	t.Run("Create with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		user := NewUserBuilder().
			WithEmail("cancelled@example.com").
			WithPasswordHash("hashedpassword").
			Build()

		err := repo.Create(ctx, user)
		assert.Error(t, err)
	})

	t.Run("FindByID with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := repo.FindByID(ctx, "some-id", dto.DefaultParanoidOptions())
		assert.Error(t, err)
	})

	t.Run("FindAll with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := repo.FindAll(ctx, &dto.ListUsersRequest{
			Page:  1,
			Limit: 10,
		})
		assert.Error(t, err)
	})
}

// TestDatabaseErrorHandling tests behavior when database encounters errors.
func TestDatabaseErrorHandling(t *testing.T) {
	// This test simulates database errors by closing the connection
	db := setupTestDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	t.Run("error after database close", func(t *testing.T) {
		// Close the database connection
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Close()
		require.NoError(t, err)

		user := NewUserBuilder().
			WithEmail("closed@example.com").
			WithPasswordHash("hashedpassword").
			Build()

		err = repo.Create(ctx, user)
		assert.Error(t, err)
	})
}
