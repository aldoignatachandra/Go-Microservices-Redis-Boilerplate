# GORM Best Practices

This boilerplate demonstrates production-ready GORM v2 patterns that are simple, type-safe, and easy to understand.

## Table of Contents

1. [Connection Management](#connection-management)
2. [Schema Design](#schema-design)
3. [Soft Delete Pattern](#soft-delete-pattern)
4. [Repository Pattern](#repository-pattern)
5. [Type Safety](#type-safety)
6. [Migrations](#migrations)

---

## Connection Management

### Simple Environment-Based Configuration

The connection is configured using environment variables with sensible defaults:

```go
// pkg/database/database.go
type Config struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}

func LoadConfig() *Config {
    return &Config{
        Host:     getEnv("DB_HOST", "localhost"),
        Port:     getEnv("DB_PORT", "5432"),
        User:     getEnv("DB_USER", "postgres"),
        Password: getEnv("DB_PASSWORD", "postgres"),
        DBName:   getEnv("DB_NAME", "cqrs_demo"),
        SSLMode:  getEnv("DB_SSLMODE", "disable"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

**Key Points:**

- ✅ Environment variables as the single source of truth
- ✅ Sensible defaults for development
- ✅ Clear function naming
- ✅ No complex configuration merging

### Connection Pooling

```go
// pkg/database/database.go
import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func NewConnection(config *Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        config.Host, config.Port, config.User,
        config.Password, config.DBName, config.SSLMode,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)

    return db, nil
}
```

**Best Practices:**

- Use connection pooling for better performance
- Enable query logging in development
- Set reasonable connection limits
- Export a single database instance for reuse

---

## Schema Design

### Base Model with Soft Delete Support

This boilerplate uses a base model struct for built-in soft delete (paranoid) support:

```go
// internal/domain/base.go
package domain

import (
    "time"
    "database/sql/driver"
    "errors"

    "gorm.io/gorm"
)

// Model is the base model for all entities
type Model struct {
    ID        string         `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
    CreatedAt time.Time      `gorm:"not null"`
    UpdatedAt time.Time      `gorm:"not null"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook to set creation timestamp
func (m *Model) BeforeCreate(tx *gorm.DB) error {
    now := time.Now().UTC()
    m.CreatedAt = now
    m.UpdatedAt = now
    return nil
}

// BeforeUpdate hook to set update timestamp
func (m *Model) BeforeUpdate(tx *gorm.DB) error {
    m.UpdatedAt = time.Now().UTC()
    return nil
}

// Scan implements sql.Scanner interface for DeletedAt
func (d *gorm.DeletedAt) Scan(value interface{}) error {
    return (*time.Time)(d).Scan(value)
}

// Value implements driver.Valuer interface for DeletedAt
func (d gorm.DeletedAt) Value() (driver.Value, error) {
    if d.Time.IsZero() {
        return nil, nil
    }
    return d.Time, nil
}

var ErrRecordNotFound = errors.New("record not found")
```

**Why This Pattern?**

- 🎯 Consistent base fields across all entities (ID, timestamps, deletedAt)
- 🎯 Automatic indexing on `deleted_at` for performance
- 🎯 DRY principle - don't repeat yourself
- 🎯 Type-safe with Go structs

### Entity Example

```go
// internal/domain/user.go
package domain

import (
    "gorm.io/gorm"
)

// Role represents the user role enum
type Role string

const (
    RoleAdmin Role = "ADMIN"
    RoleUser  Role = "USER"
)

// User represents a user entity
type User struct {
    Model
    Email    string `gorm:"type:varchar(255);not null;uniqueIndex"`
    Password string `gorm:"type:text;not null"`
    Role     Role   `gorm:"type:varchar(50);not null;default:'USER'"`
    Products []Product `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for User
func (User) TableName() string {
    return "users"
}

// BeforeCreate hook for User-specific logic
func (u *User) BeforeCreate(tx *gorm.DB) error {
    // Hash password before creating
    if err := u.HashPassword(); err != nil {
        return err
    }
    return u.Model.BeforeCreate(tx)
}

// HashPassword hashes the user's password
func (u *User) HashPassword() error {
    // Implementation here
    return nil
}
```

**Key Benefits:**

- ✨ Clean and concise schema definition
- ✨ Automatic type inference from structs
- ✨ No manual SQL needed for basic operations
- ✨ Hooks for lifecycle events

---

## Soft Delete Pattern

### Why Soft Delete?

Soft delete (paranoid) allows you to "delete" records without actually removing them from the database. This is useful for:

- 📊 Audit trails
- 🔄 Data recovery
- 📈 Historical analysis
- 🔐 Compliance requirements

### Implementation

```go
// internal/repository/user_repository.go
package repository

import (
    "context"
    "errors"

    "yourproject/internal/domain"
    "gorm.io/gorm"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

// Soft delete a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}

// Hard delete (permanent)
func (r *UserRepository) HardDelete(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).Unscoped().Delete(&domain.User{}, "id = ?", id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}

// Restore a deleted user
func (r *UserRepository) Restore(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).
        Unscoped().
        Model(&domain.User{}).
        Where("id = ? AND deleted_at IS NOT NULL", id).
        Update("deleted_at", nil)

    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}
```

### Querying with Soft Delete

```go
// Find only non-deleted users (default behavior)
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
    var user domain.User
    result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, domain.ErrRecordNotFound
        }
        return nil, result.Error
    }
    return &user, nil
}

// Include deleted users
func (r *UserRepository) FindByIDWithDeleted(ctx context.Context, id string) (*domain.User, error) {
    var user domain.User
    result := r.db.WithContext(ctx).
        Unscoped().
        Where("id = ?", id).
        First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, domain.ErrRecordNotFound
        }
        return nil, result.Error
    }
    return &user, nil
}

// Only deleted users
func (r *UserRepository) FindDeletedOnly(ctx context.Context) ([]domain.User, error) {
    var users []domain.User
    result := r.db.WithContext(ctx).
        Unscoped().
        Where("deleted_at IS NOT NULL").
        Find(&users)
    if result.Error != nil {
        return nil, result.Error
    }
    return users, nil
}
```

**Performance Note:** The `deleted_at` column is automatically indexed for efficient queries.

---

## Repository Pattern

### Why Repository Pattern?

The repository pattern provides:

- 🏗️ Abstraction over database operations
- 🧪 Easier testing (can mock repositories)
- 🔄 Consistent API across entities
- 📦 Encapsulation of query logic

### Example Repository

```go
// internal/repository/user_repository.go
package repository

import (
    "context"
    "errors"

    "yourproject/internal/domain"
    "gorm.io/gorm"
)

type UserRepository interface {
    FindByID(ctx context.Context, id string) (*domain.User, error)
    FindByEmail(ctx context.Context, email string) (*domain.User, error)
    FindAll(ctx context.Context) ([]domain.User, error)
    Create(ctx context.Context, user *domain.User) error
    Update(ctx context.Context, user *domain.User) error
    Delete(ctx context.Context, id string) error
    Restore(ctx context.Context, id string) error
}

type gormUserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &gormUserRepository{db: db}
}

func (r *gormUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
    var user domain.User
    result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, domain.ErrRecordNotFound
        }
        return nil, result.Error
    }
    return &user, nil
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
    var user domain.User
    result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, domain.ErrRecordNotFound
        }
        return nil, result.Error
    }
    return &user, nil
}

func (r *gormUserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
    var users []domain.User
    result := r.db.WithContext(ctx).Find(&users)
    if result.Error != nil {
        return nil, result.Error
    }
    return users, nil
}

func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) error {
    // Check for duplicates
    existing, err := r.FindByEmail(ctx, user.Email)
    if err == nil && existing != nil {
        return errors.New("user with this email already exists")
    }

    result := r.db.WithContext(ctx).Create(user)
    if result.Error != nil {
        return result.Error
    }
    return nil
}

func (r *gormUserRepository) Update(ctx context.Context, user *domain.User) error {
    result := r.db.WithContext(ctx).Save(user)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}

func (r *gormUserRepository) Delete(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}

func (r *gormUserRepository) Restore(ctx context.Context, id string) error {
    result := r.db.WithContext(ctx).
        Unscoped().
        Model(&domain.User{}).
        Where("id = ? AND deleted_at IS NOT NULL", id).
        Update("deleted_at", nil)

    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return domain.ErrRecordNotFound
    }
    return nil
}
```

**Usage in Services:**

```go
// internal/service/user_service.go
package service

import (
    "context"
    "yourproject/internal/domain"
    "yourproject/internal/repository"
)

type UserService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
    return &UserService{
        userRepo: userRepo,
    }
}

func (s *UserService) GetUser(ctx context.Context, id string) (*domain.User, error) {
    return s.userRepo.FindByID(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, user *domain.User) error {
    return s.userRepo.Create(ctx, user)
}
```

---

## Type Safety

### Go's Type System

Go provides strong static typing:

```go
// Strong typing with Go structs
type User struct {
    ID       string `gorm:"type:uuid;primary_key"`
    Email    string `gorm:"type:varchar(255);not null;uniqueIndex"`
    Password string `gorm:"type:text;not null"`
    Role     Role   `gorm:"type:varchar(50);not null;default:'USER'"`
}

// Enum-like type for roles
type Role string

const (
    RoleAdmin Role = "ADMIN"
    RoleUser  Role = "USER"
)

// Type-safe query builder
func (r *gormUserRepository) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
    var users []domain.User
    result := r.db.WithContext(ctx).Where("role = ?", role).Find(&users)
    if result.Error != nil {
        return nil, result.Error
    }
    return users, nil
}
```

### Custom Types

```go
// Nullable types
type NullableString struct {
    String string
    Valid  bool
}

// Custom scanner
func (ns *NullableString) Scan(value interface{}) error {
    if value == nil {
        ns.String, ns.Valid = "", false
        return nil
    }
    ns.Valid = true
    return nil
}

// Custom value driver
func (ns NullableString) Value() (driver.Value, error) {
    if !ns.Valid {
        return nil, nil
    }
    return ns.String, nil
}
```

**Benefits:**

- 💪 Full compile-time type safety
- 🛡️ No runtime type errors
- 🔄 Schema changes require code updates (catch errors early)
- 📝 Self-documenting code

---

## Migrations

### Running Migrations

```bash
# Run auto-migration
go run cmd/migrate/main.go

# Or using make
make migrate
```

### Migration Best Practices

1. **Always review generated migrations** before running them
2. **Test migrations on a copy of production data** before deploying
3. **Keep migrations reversible** when possible
4. **Version control all migrations**
5. **Don't modify past migrations** after they've been run in production

### Auto-Migration

```go
// cmd/migrate/main.go
package main

import (
    "log"

    "yourproject/internal/domain"
    "yourproject/pkg/database"
    "gorm.io/gorm"
)

func main() {
    config := database.LoadConfig()
    db, err := database.NewConnection(config)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    if err := runMigrations(db); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    log.Println("Migrations completed successfully")
}

func runMigrations(db *gorm.DB) error {
    // Auto-migrate all entities
    return db.AutoMigrate(
        &domain.User{},
        &domain.Product{},
    )
}
```

### Manual Migrations (for production)

For production, use a proper migration tool like [golang-migrate/migrate](https://github.com/golang-migrate/migrate):

```bash
# Create migration
migrate create -ext sql -dir migrations -seq create_users_table

# Run migration
migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

# Rollback
migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down 1
```

Example SQL migration file:

```sql
-- migrations/000001_create_users_table.up.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- migrations/000001_create_users_table.down.sql
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;
```

---

## Summary

This boilerplate follows these key principles:

1. **Simplicity First** - No over-engineering
2. **Type Safety** - Leverage Go's strong typing
3. **Environment-Based Config** - Single source of truth
4. **Soft Delete by Default** - Safer than hard deletes
5. **Repository Pattern** - Clean separation of concerns
6. **Educational** - Easy to learn and understand

### Quick Reference

| Task              | Command                      |
| ----------------- | ---------------------------- |
| Run migrations    | `go run cmd/migrate/main.go` |
| Run tests         | `go test ./... -race`        |
| Run with coverage | `go test ./... -cover -race` |
| Format code       | `goimports -w .`             |
| Run linters       | `golangci-lint run ./...`    |
| Build             | `go build ./...`             |
| Run               | `go run cmd/service/main.go` |

### Further Reading

- [GORM Documentation](https://gorm.io/docs/)
- [PostgreSQL Best Practices](https://wiki.postgresql.org/wiki/Don%27t_Do_This)
- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
- [Go Database SQL Tutorial](https://go.dev/doc/database/index)
