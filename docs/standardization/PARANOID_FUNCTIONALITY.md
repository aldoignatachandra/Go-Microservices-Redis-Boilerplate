# Paranoid Functionality Implementation

This document describes the comprehensive paranoid (soft delete) functionality implemented across both user and product services.

## Overview

The paranoid functionality provides soft delete capabilities with the following features:

- Soft delete by default (sets `deletedAt` timestamp)
- Hard delete option (permanent deletion)
- Restore soft-deleted records
- Query filtering based on deletion status
- Comprehensive error handling
- Enhanced API responses with paranoid metadata

## Architecture

### Core Components

1. **Base Model with Paranoid Support** (`internal/domain/base.go`)
   - Implements paranoid fields (ID, timestamps, deletedAt)
   - Provides `gorm.DeletedAt` for soft delete
   - Handles lifecycle hooks for timestamps

2. **Repository Interface** (`internal/repository/user_repository.go`)
   - Implements paranoid query building
   - Provides `Unscoped()` for including deleted records
   - Handles soft delete and restore operations

3. **Enhanced Error Handling** (`internal/errors/paranoid_errors.go`)
   - `ErrRecordNotFound` base error
   - `ErrSoftDeleted` error for deleted records
   - Validation functions for paranoid operations

4. **Response Type System** (`internal/dto/paranoid_responses.go`)
   - Standardized response formats
   - Paranoid metadata inclusion
   - Operation result tracking

## API Endpoints

### User Service (`cmd/auth-service`, `cmd/user-service`)

#### Authentication Routes

- `POST /login` - Login (includes deleted records for authentication)
- `GET /me` - Get current user (active only)

#### Admin Routes

- `GET /admin/users` - List users with paranoid options
  - Query parameters:
    - `include_deleted=true|false` - Include soft-deleted records
    - `only_deleted=true|false` - Only deleted records
    - `role=ADMIN|USER` - Filter by role
- `GET /admin/users/:id` - Get user by ID with paranoid options
  - Query parameters:
    - `include_deleted=true|false` - Include soft-deleted records
- `DELETE /admin/users/:id` - Delete user (soft by default)
  - Query parameters:
    - `force=true|false` - Hard delete when true
- `POST /admin/users/:id/restore` - Restore soft-deleted user

### Product Service (`cmd/product-service`)

#### Product Routes

- `POST /products` - Create product
- `GET /products` - List user's products with paranoid options
  - Query parameters:
    - `include_deleted=true|false` - Include soft-deleted records
    - `only_deleted=true|false` - Only deleted records
    - `search=query` - Search by name
    - `min_price=number` - Minimum price filter
    - `max_price=number` - Maximum price filter
- `GET /products/:id` - Get product by ID with paranoid options
  - Query parameters:
    - `include_deleted=true|false` - Include soft-deleted records
- `PATCH /products/:id` - Update product (owner only)
- `DELETE /products/:id` - Delete product (soft by default)
  - Query parameters:
    - `force=true|false` - Hard delete when true
- `POST /products/:id/restore` - Restore soft-deleted product
- `GET /products/search` - Search products with paranoid options

## Query Options

### Paranoid Options

```go
// internal/dto/paranoid_options.go
package dto

type ParanoidOptions struct {
    IncludeDeleted bool `json:"include_deleted"` // Include both active and deleted records
    OnlyDeleted    bool `json:"only_deleted"`    // Only deleted records
    OnlyActive     bool `json:"only_active"`     // Only active records (default)
}
```

### Validation Rules

1. **Mutually Exclusive**: Only one of `IncludeDeleted`, `OnlyDeleted`, or `OnlyActive` can be true
2. **Default Behavior**: When no option is specified, `OnlyActive: true` is assumed
3. **Error Handling**: Invalid combinations throw validation error

```go
// internal/dto/paranoid_options.go
func (p *ParanoidOptions) Validate() error {
    count := 0
    if p.IncludeDeleted {
        count++
    }
    if p.OnlyDeleted {
        count++
    }
    if p.OnlyActive {
        count++
    }

    if count > 1 {
        return errors.New("only one paranoid option can be true")
    }

    return nil
}
```

## Repository Methods

### Enhanced Methods

```go
// internal/repository/user_repository.go
type UserRepository interface {
    FindByID(ctx context.Context, id string, opts *dto.ParanoidOptions) (*domain.User, error)
    FindByEmail(ctx context.Context, email string, opts *dto.ParanoidOptions) (*domain.User, error)
    FindAll(ctx context.Context, opts *dto.ParanoidOptions) ([]domain.User, error)
    Delete(ctx context.Context, id string, force bool) error
    Restore(ctx context.Context, id string) error
}

// internal/repository/product_repository.go
type ProductRepository interface {
    FindByID(ctx context.Context, id string, opts *dto.ParanoidOptions) (*domain.Product, error)
    FindByOwner(ctx context.Context, ownerID string, opts *dto.ParanoidOptions) ([]domain.Product, error)
    Delete(ctx context.Context, id string, force bool) error
    Restore(ctx context.Context, id string) error
}
```

### Specialized Methods

```go
// Include deleted records
func (r *gormUserRepository) FindByIDWithDeleted(ctx context.Context, id string) (*domain.User, error) {
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

// Only deleted records
func (r *gormUserRepository) FindDeletedOnly(ctx context.Context) ([]domain.User, error) {
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

// Search with paranoid options
func (r *gormProductRepository) SearchWithDeleted(ctx context.Context, query string) ([]domain.Product, error) {
    var products []domain.Product
    result := r.db.WithContext(ctx).
        Unscoped().
        Where("name ILIKE ?", "%"+query+"%").
        Find(&products)

    if result.Error != nil {
        return nil, result.Error
    }

    return products, nil
}

func (r *gormProductRepository) SearchDeletedOnly(ctx context.Context, query string) ([]domain.Product, error) {
    var products []domain.Product
    result := r.db.WithContext(ctx).
        Unscoped().
        Where("deleted_at IS NOT NULL").
        Where("name ILIKE ?", "%"+query+"%").
        Find(&products)

    if result.Error != nil {
        return nil, result.Error
    }

    return products, nil
}
```

## Error Handling

### Error Types

```go
// internal/errors/paranoid_errors.go
package errors

import "fmt"

var (
    // ErrRecordNotFound is returned when a record is not found
    ErrRecordNotFound = fmt.Errorf("record not found")

    // ErrSoftDeleted is returned when attempting to access a soft-deleted record
    ErrSoftDeleted = fmt.Errorf("record is soft deleted")

    // ErrAccessDenied is returned for insufficient permissions
    ErrAccessDenied = fmt.Errorf("access denied")

    // ErrInvalidParanoidOptions is returned for invalid query parameter combinations
    ErrInvalidParanoidOptions = fmt.Errorf("invalid paranoid options")
)
```

### Error Response Format

```go
// internal/dto/error_response.go
package dto

type ErrorResponse struct {
    Error ErrorDetails `json:"error"`
    Meta  ResponseMeta `json:"meta"`
}

type ErrorDetails struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details *struct {
        Resource string            `json:"resource,omitempty"`
        ID       string            `json:"id,omitempty"`
        Paranoid *ParanoidOptions  `json:"paranoid,omitempty"`
    } `json:"details,omitempty"`
}

type ResponseMeta struct {
    Timestamp string `json:"timestamp"`
    RequestID string `json:"request_id,omitempty"`
}
```

## Response Formats

### Standard Response

```go
// internal/dto/response.go
package dto

import "time"

type Response[T any] struct {
    Data T `json:"data"`
    Meta ResponseMeta `json:"meta"`
}

type ResponseMeta struct {
    Paranoid   *ParanoidOptions  `json:"paranoid,omitempty"`
    Pagination *PaginationMeta   `json:"pagination,omitempty"`
    Timestamp  string            `json:"timestamp"`
    RequestID  string            `json:"request_id,omitempty"`
}

type PaginationMeta struct {
    Page       int  `json:"page"`
    Limit      int  `json:"limit"`
    Total      int64 `json:"total"`
    TotalPages int  `json:"total_pages"`
    HasNext    bool `json:"has_next"`
    HasPrev    bool `json:"has_prev"`
}
```

### List Response

```go
type ListResponse[T any] struct {
    Data []T `json:"data"`
    Meta ListResponseMeta `json:"meta"`
}

type ListResponseMeta struct {
    Paranoid   *ParanoidOptions  `json:"paranoid,omitempty"`
    Pagination *PaginationMeta   `json:"pagination,omitempty"`
    Count      int               `json:"count"`
    Filters    map[string]string `json:"filters,omitempty"`
    Timestamp  string            `json:"timestamp"`
    RequestID  string            `json:"request_id,omitempty"`
}
```

## Event Publishing

### User Events

```go
// internal/events/user_events.go
package events

const (
    UserCreatedEventType   = "user.created"
    UserRestoredEventType  = "user.restored"
)

type UserEvent struct {
    EventType string      `json:"event_type"`
    UserID    string      `json:"user_id"`
    Timestamp time.Time   `json:"timestamp"`
    Metadata  interface{} `json:"metadata,omitempty"`
}
```

### Product Events

```go
// internal/events/product_events.go
package events

const (
    ProductCreatedEventType  = "product.created"
    ProductUpdatedEventType  = "product.updated"
    ProductDeletedEventType  = "product.deleted"
    ProductRestoredEventType = "product.restored"
)

type ProductEvent struct {
    EventType  string      `json:"event_type"`
    ProductID  string      `json:"product_id"`
    OwnerID    string      `json:"owner_id"`
    Timestamp  time.Time   `json:"timestamp"`
    Metadata   interface{} `json:"metadata,omitempty"`
}
```

## Usage Examples

### Basic Queries

```go
// Only active records (default)
users, err := userRepo.FindAll(ctx, &dto.ParanoidOptions{OnlyActive: true})

// Include deleted records
allUsers, err := userRepo.FindAll(ctx, &dto.ParanoidOptions{IncludeDeleted: true})

// Only deleted records
deletedUsers, err := userRepo.FindAll(ctx, &dto.ParanoidOptions{OnlyDeleted: true})
```

### Search with Filters

```go
// Search active products
products, err := productRepo.Search(ctx, "laptop", &dto.ParanoidOptions{OnlyActive: true})

// Search including deleted
allProducts, err := productRepo.SearchWithDeleted(ctx, "laptop")

// Search only deleted
deletedProducts, err := productRepo.SearchDeletedOnly(ctx, "laptop")
```

### Restore Operations

```go
// Restore user
err := userRepo.Restore(ctx, "user-id")

// Restore product
err := productRepo.Restore(ctx, "product-id")
```

## Testing

Comprehensive test suite is available covering:

- Repository method behavior
- Query parameter validation
- Error handling scenarios
- Response format validation
- API endpoint behavior
- Integration scenarios

### Example Test

```go
// internal/repository/user_repository_test.go
package repository_test

import (
    "context"
    "testing"

    "yourproject/internal/domain"
    "yourproject/internal/dto"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_ParanoidOperations(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    repo := NewUserRepository(db)
    ctx := context.Background()

    // Create test user
    user := &domain.User{
        Email:    "test@example.com",
        Password: "hashed",
        Role:     domain.RoleUser,
    }
    err := repo.Create(ctx, user)
    require.NoError(t, err)

    // Test soft delete
    err = repo.Delete(ctx, user.ID, false)
    require.NoError(t, err)

    // Verify user is soft deleted (not found with default query)
    _, err = repo.FindByID(ctx, user.ID, &dto.ParanoidOptions{OnlyActive: true})
    assert.Error(t, err)

    // Verify user can be found with paranoid options
    found, err := repo.FindByID(ctx, user.ID, &dto.ParanoidOptions{IncludeDeleted: true})
    require.NoError(t, err)
    assert.Equal(t, user.ID, found.ID)

    // Test restore
    err = repo.Restore(ctx, user.ID)
    require.NoError(t, err)

    // Verify user is restored
    found, err = repo.FindByID(ctx, user.ID, &dto.ParanoidOptions{OnlyActive: true})
    require.NoError(t, err)
    assert.Equal(t, user.ID, found.ID)

    // Test hard delete
    err = repo.Delete(ctx, user.ID, true)
    require.NoError(t, err)

    // Verify user is permanently deleted
    _, err = repo.FindByIDWithDeleted(ctx, user.ID)
    assert.Error(t, err)
}
```

Run tests with:

```bash
go test ./internal/repository/... -v -race
```

## Migration Guide

### For Existing APIs

1. **Add Paranoid Options**: Update method signatures to accept `*dto.ParanoidOptions`
2. **Update Queries**: Use `Unscoped()` for including deleted records
3. **Handle Soft Deletes**: Use `Delete(id, false)` instead of hard deletes
4. **Add Restore Endpoints**: Implement restore functionality where needed
5. **Update Error Handling**: Use new error types and response formats

### Database Considerations

1. **Index Strategy**: Consider adding indexes on `deleted_at` for performance
2. **Query Optimization**: Use appropriate where clauses for deleted filtering
3. **Data Integrity**: Ensure foreign key constraints handle soft deletes properly

## Best Practices

1. **Default to Active**: Always exclude deleted records unless explicitly requested
2. **Validate Options**: Check for mutually exclusive paranoid options
3. **Consistent Responses**: Use standardized response formats with paranoid metadata
4. **Proper Error Codes**: Use appropriate HTTP status codes (410 for soft deleted)
5. **Event Publishing**: Emit appropriate events for restore operations
6. **Ownership Checks**: Verify ownership for restore operations
7. **Audit Logging**: Log all restore and hard delete operations

## Security Considerations

1. **Access Control**: Restore operations should require appropriate permissions
2. **Audit Trail**: Maintain audit logs for restore operations
3. **Data Privacy**: Ensure deleted data is properly handled per privacy requirements
4. **Rate Limiting**: Consider rate limiting restore operations
5. **Validation**: Validate restore requests to prevent unauthorized access

## Performance Considerations

1. **Query Optimization**: Use appropriate indexes for `deleted_at` queries
2. **Pagination**: Implement proper pagination for large result sets
3. **Caching**: Consider caching strategies for frequently accessed active records
4. **Batch Operations**: Use batch operations for bulk restore/delete operations
5. **Monitoring**: Monitor performance of paranoid queries

---

**Note**: This implementation uses GORM v2's built-in soft delete functionality with `gorm.DeletedAt`. The `Unscoped()` method is used to bypass soft delete filtering when needed.
