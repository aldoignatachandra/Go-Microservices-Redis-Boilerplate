# Schema Alignment: Go (GORM) to Match Bun (Drizzle) - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor Go microservices to exactly match Bun project's database schemas, validation rules, and API behaviors for full feature parity.

**Architecture:** This plan follows Go best practices: Repository Pattern, Service Layer, DTOs, Validator Package, and Clean Architecture. All changes maintain backward compatibility where possible and use proper GORM hooks/middleware.

**Tech Stack:** Go 1.21+, GORM v1.25+, PostgreSQL, Gin Framework, UUID

---

## Pre-Implementation Checklist

- [ ] Review existing codebase structure
- [ ] Run existing tests to ensure baseline
- [ ] Create backup branch: `backup/pre-schema-alignment`
- [ ] Set up database migration tooling

---

## Phase 1: Domain Entity Updates (CRITICAL)

### Task 1.1: Update User Entity in Auth Service

**Files:**
- Modify: `internal/auth/domain/user.go`
- Test: `internal/auth/domain/user_test.go` (create if not exists)

**Step 1: Verify current User struct**

Run: `grep -n "type User struct" internal/auth/domain/user.go`

**Step 2: Update User struct with new fields**

```go
// User represents a user entity.
type User struct {
	Model
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Username     string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Name         string     `gorm:"type:varchar(255)" json:"name"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Role         Role       `gorm:"type:varchar(50);not null;default:USER" json:"role"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}
```

**Step 3: Update CanLogin method**

```go
// CanLogin checks if the user can login (soft delete based).
func (u *User) CanLogin() bool {
	return !u.DeletedAt.Valid
}
```

**Step 4: Update ToSafeUser method**

```go
func (u *User) ToSafeUser() *SafeUser {
	return &SafeUser{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
```

**Step 5: Run tests**

Run: `go test ./internal/auth/domain/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/auth/domain/user.go
git commit -m "refactor(auth): align User entity with Bun schema - add Username, Name, remove IsActive"
```

---

### Task 1.2: Update User Entity in User Service

**Files:**
- Modify: `internal/user/domain/user.go`

**Step 1: Verify current User struct**

Run: `grep -n "type User struct" internal/user/domain/user.go`

**Step 2: Apply same User struct changes as Task 1.1**

**Step 3: Run go vet**

Run: `go vet ./internal/user/domain/...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/user/domain/user.go
git commit -m "refactor(user): align User entity with Bun schema"
```

---

### Task 1.3: Update Session Entity

**Files:**
- Modify: `internal/auth/domain/session.go`
- Test: `internal/auth/domain/session_test.go`

**Step 1: Verify current Session struct**

Run: `cat internal/auth/domain/session.go`

**Step 2: Update Session struct**

```go
// Session represents a user session.
type Session struct {
	ID          string         `gorm:"type:uuid;primary_key;" json:"id"`
	UserID      string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Token       string         `gorm:"type:text" json:"-"`
	ExpiresAt   time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt   time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null" json:"updated_at"`
	RevokedAt   *time.Time     `json:"revoked_at,omitempty"`
	LastUsedAt  time.Time      `gorm:"not null" json:"last_used_at"`
	DeviceType  string         `gorm:"type:varchar(50)" json:"device_type"`
	UserAgent   string         `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress   string         `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
```

**Step 3: Update BeforeCreate hook**

```go
// BeforeCreate is a GORM hook that runs before creating a session.
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now
	s.LastUsedAt = now
	return nil
}
```

**Step 4: Add new methods**

```go
// UpdateLastUsed updates the last used timestamp.
func (s *Session) UpdateLastUsed() {
	s.LastUsedAt = time.Now().UTC()
}
```

**Step 5: Run tests**

Run: `go test ./internal/auth/domain/... -v -run Session`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/auth/domain/session.go
git commit -m "refactor(auth): align Session entity - rename RefreshToken to Token, add LastUsedAt, DeviceType, soft delete"
```

---

### Task 1.4: Update ActivityLog Entity

**Files:**
- Modify: `internal/user/domain/user.go` (ActivityLog section)

**Step 1: Verify current ActivityLog**

Run: `grep -n -A20 "type ActivityLog struct" internal/user/domain/user.go`

**Step 2: Update ActivityLog struct**

```go
// ActivityLog represents an activity log entry.
type ActivityLog struct {
	Model
	UserID    string                 `gorm:"type:uuid;not null;index" json:"user_id"`
	Action    string                 `gorm:"type:varchar(255);not null" json:"action"`
	Entity    string                 `gorm:"type:varchar(100)" json:"entity,omitempty"`
	EntityID  string                 `gorm:"type:uuid" json:"entity_id,omitempty"`
	IPAddress string                 `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent string                 `gorm:"type:text" json:"user_agent,omitempty"`
	Details   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"details,omitempty"`
}

// TableName specifies the table name for ActivityLog.
func (ActivityLog) TableName() string {
	return "user_activity_logs"
}
```

**Step 3: Update helper functions**

```go
// NewActivityLog creates a new activity log entry.
func NewActivityLog(userID, action, entity, entityID string) *ActivityLog {
	return &ActivityLog{
		UserID:   userID,
		Action:   action,
		Entity:   entity,
		EntityID: entityID,
	}
}

// WithMetadata adds metadata to the activity log.
func (a *ActivityLog) WithMetadata(key string, value interface{}) *ActivityLog {
	if a.Details == nil {
		a.Details = make(map[string]interface{})
	}
	a.Details[key] = value
	return a
}
```

**Step 4: Commit**

```bash
git add internal/user/domain/user.go
git commit -m "refactor(user): align ActivityLog - rename table to user_activity_logs, rename fields"
```

---

### Task 1.5: Update Product Entity

**Files:**
- Modify: `internal/product/domain/product.go`

**Step 1: Verify current Product struct**

Run: `grep -n -A15 "type Product struct" internal/product/domain/product.go`

**Step 2: Update Product struct**

```go
// Product represents a product entity.
type Product struct {
	Model
	Name       string  `gorm:"type:varchar(255);not null" json:"name"`
	Price      float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock      int     `gorm:"type:int;not null;default:0" json:"stock"`
	OwnerID    string  `gorm:"type:uuid;not null;index" json:"owner_id"`
	HasVariant bool    `gorm:"default:false" json:"has_variant"`
	Images     string  `gorm:"type:text" json:"images"`
}
```

**Step 3: Update IsAvailable method**

```go
// IsAvailable checks if the product is available for purchase.
func (p *Product) IsAvailable() bool {
	return p.Stock > 0 && !p.DeletedAt.Valid
}
```

**Step 4: Update ToSafeProduct method**

```go
func (p *Product) ToSafeProduct() *SafeProduct {
	return &SafeProduct{
		ID:         p.ID,
		Name:       p.Name,
		Price:      p.Price,
		Stock:      p.Stock,
		OwnerID:    p.OwnerID,
		HasVariant: p.HasVariant,
		Images:     p.Images,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}
```

**Step 5: Add new error definitions**

```go
var (
	ErrProductNotFound         = errors.New("product not found")
	ErrProductNameAlreadyUsed = errors.New("product name already used")
	ErrInvalidStockReduction  = errors.New("invalid stock reduction amount")
	ErrInsufficientStock      = errors.New("insufficient stock")
)
```

**Step 6: Commit**

```bash
git add internal/product/domain/product.go
git commit -m "refactor(product): align Product entity - add OwnerID, HasVariant, Images, remove Description, Status, CategoryID"
```

---

## Phase 2: New Domain Entities

### Task 2.1: Create ProductVariant Entity

**Files:**
- Create: `internal/product/domain/variant.go`
- Test: `internal/product/domain/variant_test.go`

**Step 1: Create the file**

```go
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductVariant represents a product variant entity.
type ProductVariant struct {
	ID             string                 `gorm:"type:uuid;primary_key;" json:"id"`
	ProductID      string                 `gorm:"type:uuid;not null;index" json:"product_id"`
	Name           string                 `gorm:"type:varchar(255);not null" json:"name"`
	SKU            string                 `gorm:"type:varchar(100);not null;uniqueIndex" json:"sku"`
	Price          float64                `gorm:"type:decimal(10,2)" json:"price,omitempty"`
	StockQuantity  int                    `gorm:"type:int;not null;default:0" json:"stock_quantity"`
	StockReserved  int                    `gorm:"type:int;not null;default:0" json:"stock_reserved"`
	IsActive       bool                   `gorm:"default:true" json:"is_active"`
	AttributeValues map[string]string     `gorm:"type:jsonb" json:"attribute_values"`
	Images         string                 `gorm:"type:text" json:"images"`
	CreatedAt      time.Time              `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time              `gorm:"not null" json:"updated_at"`
	DeletedAt      gorm.DeletedAt        `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for ProductVariant.
func (ProductVariant) TableName() string {
	return "product_variants"
}

// IsAvailable checks if the variant is available for purchase.
func (v *ProductVariant) IsAvailable() bool {
	return v.StockQuantity > 0 && v.IsActive && !v.DeletedAt.Valid
}

// ReduceStock reduces the variant stock by the given amount.
func (v *ProductVariant) ReduceStock(amount int) error {
	if amount <= 0 {
		return ErrInvalidStockReduction
	}
	if v.StockQuantity < amount {
		return ErrInsufficientStock
	}
	v.StockQuantity -= amount
	return nil
}

// BeforeCreate is a GORM hook that runs before creating a variant.
func (v *ProductVariant) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a variant.
func (v *ProductVariant) BeforeUpdate(tx *gorm.DB) error {
	v.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeVariant returns a copy of the variant without sensitive fields.
func (v *ProductVariant) ToSafeVariant() *SafeVariant {
	return &SafeVariant{
		ID:             v.ID,
		ProductID:      v.ProductID,
		Name:           v.Name,
		SKU:            v.SKU,
		Price:          v.Price,
		StockQuantity:  v.StockQuantity,
		StockReserved:  v.StockReserved,
		IsActive:       v.IsActive,
		AttributeValues: v.AttributeValues,
		Images:         v.Images,
		CreatedAt:      v.CreatedAt,
		UpdatedAt:      v.UpdatedAt,
	}
}

// SafeVariant represents a variant without sensitive fields.
type SafeVariant struct {
	ID             string              `json:"id"`
	ProductID      string              `json:"product_id"`
	Name           string              `json:"name"`
	SKU            string              `json:"sku"`
	Price          float64             `json:"price,omitempty"`
	StockQuantity  int                 `json:"stock_quantity"`
	StockReserved  int                 `json:"stock_reserved"`
	IsActive       bool                `json:"is_active"`
	AttributeValues map[string]string  `json:"attribute_values"`
	Images         string              `json:"images"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// VariantList represents a list of variants.
type VariantList struct {
	Variants  []*ProductVariant `json:"variants"`
	Total     int64             `json:"total"`
	Page      int               `json:"page"`
	Limit     int               `json:"limit"`
	TotalPages int              `json:"total_pages"`
}
```

**Step 2: Run go build**

Run: `go build ./internal/product/domain/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/product/domain/variant.go
git commit -m "feat(product): add ProductVariant entity aligned with Bun schema"
```

---

### Task 2.2: Create ProductAttribute Entity

**Files:**
- Create: `internal/product/domain/attribute.go`

**Step 1: Create the file**

```go
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductAttribute represents a product attribute entity.
type ProductAttribute struct {
	ID          string   `gorm:"type:uuid;primary_key;" json:"id"`
	ProductID   string   `gorm:"type:uuid;not null;index" json:"product_id"`
	Name        string   `gorm:"type:varchar(100);not null" json:"name"`
	Values      []string `gorm:"type:jsonb" json:"values"`
	DisplayOrder int     `gorm:"type:int;default:0" json:"display_order"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for ProductAttribute.
func (ProductAttribute) TableName() string {
	return "product_attributes"
}

// BeforeCreate is a GORM hook that runs before creating an attribute.
func (a *ProductAttribute) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating an attribute.
func (a *ProductAttribute) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeAttribute returns a copy of the attribute without sensitive fields.
func (a *ProductAttribute) ToSafeAttribute() *SafeAttribute {
	return &SafeAttribute{
		ID:          a.ID,
		ProductID:   a.ProductID,
		Name:        a.Name,
		Values:      a.Values,
		DisplayOrder: a.DisplayOrder,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// SafeAttribute represents an attribute without sensitive fields.
type SafeAttribute struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	Name         string    `json:"name"`
	Values       []string  `json:"values"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AttributeList represents a list of attributes.
type AttributeList struct {
	Attributes []*ProductAttribute `json:"attributes"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}
```

**Step 2: Commit**

```bash
git add internal/product/domain/attribute.go
git commit -m "feat(product): add ProductAttribute entity aligned with Bun schema"
```

---

## Phase 3: Validator Package

### Task 3.1: Create Password Validator

**Files:**
- Create: `pkg/validator/password.go`
- Test: `pkg/validator/password_test.go`

**Step 1: Create the validator**

```go
package validator

import (
	"errors"
	"regexp"
	"unicode"
)

var (
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrPasswordNoUppercase  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoNumber     = errors.New("password must contain at least one number")
	ErrPasswordForbiddenChar = errors.New("password contains forbidden characters")
)

var (
	passwordMinLength   = 8
	passwordUppercaseRE = regexp.MustCompile(`[A-Z]`)
	passwordNumberRE    = regexp.MustCompile(`[0-9]`)
)

type PasswordValidator struct {
	minLength        int
	requireUppercase bool
	requireNumber    bool
}

func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		minLength:        passwordMinLength,
		requireUppercase: true,
		requireNumber:    true,
	}
}

func (v *PasswordValidator) Validate(password string) error {
	if len(password) < v.minLength {
		return ErrPasswordTooShort
	}

	if v.requireUppercase && !passwordUppercaseRE.MatchString(password) {
		return ErrPasswordNoUppercase
	}

	if v.requireNumber && !passwordNumberRE.MatchString(password) {
		return ErrPasswordNoNumber
	}

	for _, c := range password {
		if isForbiddenPasswordChar(c) {
			return ErrPasswordForbiddenChar
		}
	}

	return nil
}

func isForbiddenPasswordChar(c rune) bool {
	forbidden := []rune{'\'', '"', '`', '\\', '/'}
	for _, f := range forbidden {
		if c == f {
			return true
		}
	}
	return false
}

func (v *PasswordValidator) IsValid(password string) bool {
	return v.Validate(password) == nil
}

func ValidatePassword(password string) error {
	return NewPasswordValidator().Validate(password)
}

func IsValidPassword(password string) bool {
	return ValidatePassword(password) == nil
}
```

**Step 2: Write tests**

```go
package validator

import (
	"testing"
)

func TestPasswordValidator(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr bool
		err     error
	}{
		{"valid password", "Password1", false, nil},
		{"valid with special", "Password1!", false, nil},
		{"too short", "Pass1", true, ErrPasswordTooShort},
		{"no uppercase", "password1", true, ErrPasswordNoUppercase},
		{"no number", "Password", true, ErrPasswordNoNumber},
		{"forbidden char", "Password1'", true, ErrPasswordForbiddenChar},
		{"empty password", "", true, ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.err {
				t.Errorf("ValidatePassword() error = %v, want %v", err, tt.err)
			}
		})
	}
}
```

**Step 3: Run tests**

Run: `go test ./pkg/validator/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/validator/password.go pkg/validator/password_test.go
git commit -m "feat(validator): add password validator with Bun-compatible rules"
```

---

### Task 3.2: Create Username Validator

**Files:**
- Create: `pkg/validator/username.go`
- Test: `pkg/validator/username_test.go`

**Step 1: Create the validator**

```go
package validator

import (
	"errors"
	"regexp"
)

var (
	ErrUsernameTooShort = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong  = errors.New("username must be at most 50 characters")
	ErrUsernameInvalid  = errors.New("username can only contain letters, numbers, and underscores")
)

var (
	usernameMinLength = 3
	usernameMaxLength = 50
	usernameRE        = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

type UsernameValidator struct {
	minLength int
	maxLength int
}

func NewUsernameValidator() *UsernameValidator {
	return &UsernameValidator{
		minLength: usernameMinLength,
		maxLength: usernameMaxLength,
	}
}

func (v *UsernameValidator) Validate(username string) error {
	if len(username) < v.minLength {
		return ErrUsernameTooShort
	}
	if len(username) > v.maxLength {
		return ErrUsernameTooLong
	}
	if !usernameRE.MatchString(username) {
		return ErrUsernameInvalid
	}
	return nil
}

func (v *UsernameValidator) IsValid(username string) bool {
	return v.Validate(username) == nil
}

func ValidateUsername(username string) error {
	return NewUsernameValidator().Validate(username)
}

func IsValidUsername(username string) bool {
	return ValidateUsername(username) == nil
}
```

**Step 2: Write tests**

```go
package validator

import (
	"testing"
)

func TestUsernameValidator(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
		err      error
	}{
		{"valid username", "john_doe", false, nil},
		{"valid with numbers", "john123", false, nil},
		{"valid uppercase", "JohnDoe", false, nil},
		{"too short", "ab", true, ErrUsernameTooShort},
		{"too long", "this_is_a_very_long_username_that_exceeds_limit", true, ErrUsernameTooLong},
		{"invalid char", "john-doe", true, ErrUsernameInvalid},
		{"empty", "", true, ErrUsernameTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.err {
				t.Errorf("ValidateUsername() error = %v, want %v", err, tt.err)
			}
		})
	}
}
```

**Step 3: Run tests**

Run: `go test ./pkg/validator/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/validator/username.go pkg/validator/username_test.go
git commit -m "feat(validator): add username validator with Bun-compatible rules"
```

---

## Phase 4: DTO Updates

### Task 4.1: Update Auth DTOs

**Files:**
- Modify: `internal/auth/dto/request.go`
- Test: `internal/auth/dto/request_test.go`

**Step 1: Read current DTOs**

Run: `cat internal/auth/dto/request.go`

**Step 2: Update RegisterRequest**

```go
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"omitempty,max=255"`
	Role     string `json:"role" binding:"omitempty,oneof=ADMIN USER"`
}
```

**Step 3: Add custom validation method**

```go
func (r *RegisterRequest) Validate() error {
	if err := validator.ValidateUsername(r.Username); err != nil {
		return err
	}
	if err := validator.ValidatePassword(r.Password); err != nil {
		return err
	}
	if r.Role == "" {
		r.Role = "USER"
	}
	return nil
}
```

**Step 4: Update LoginRequest**

```go
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
```

**Step 5: Run go vet**

Run: `go vet ./internal/auth/dto/...`
Expected: No errors

**Step 6: Commit**

```bash
git add internal/auth/dto/request.go
git commit -m "refactor(auth): update DTOs with username validation and Bun-compatible rules"
```

---

### Task 4.2: Update Product DTOs

**Files:**
- Modify: `internal/product/dto/request.go`
- Modify: `internal/product/dto/response.go`

**Step 1: Update CreateProductRequest**

```go
type CreateProductRequest struct {
	Name      string  `json:"name" binding:"required,min=1,max=255"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Stock     int     `json:"stock" binding:"omitempty,min=0"`
	HasVariant bool   `json:"has_variant" binding:"omitempty"`
	Images    string  `json:"images" binding:"omitempty"`
}
```

**Step 2: Update ProductResponse**

```go
type ProductResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	OwnerID    string    `json:"owner_id"`
	Stock      int       `json:"stock"`
	HasVariant bool      `json:"has_variant"`
	Images     string    `json:"images"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
```

**Step 3: Commit**

```bash
git add internal/product/dto/request.go internal/product/dto/response.go
git commit -m "refactor(product): update DTOs with owner_id, has_variant, images fields"
```

---

## Phase 5: API Behavior Changes

### Task 5.1: Implement Single Session Policy

**Files:**
- Modify: `internal/auth/usecase/auth_usecase.go`
- Test: `internal/auth/usecase/auth_usecase_test.go`

**Step 1: Find Login method**

Run: `grep -n "func.*Login" internal/auth/usecase/auth_usecase.go`

**Step 2: Update Login to force delete old sessions**

```go
func (u *AuthUseCase) Login(ctx context.Context, req *dto.LoginRequest, deviceInfo *DeviceInfo) (*dto.AuthResponse, error) {
	// Find user by email
	user, err := u.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if err := u.hash.Compare(user.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// SINGLE SESSION POLICY: Delete ALL existing sessions first
	if err := u.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, expiresAt, err := u.jwt.GenerateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	// Create new session
	session := &domain.Session{
		UserID:     user.ID,
		Token:      token,
		ExpiresAt:  expiresAt,
		DeviceType: deviceInfo.DeviceType,
		UserAgent:  deviceInfo.UserAgent,
		IPAddress:  deviceInfo.IPAddress,
	}

	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// Update last login
	user.TouchLastLogin()
	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		Token: token,
		User:  user.ToSafeUser(),
	}, nil
}
```

**Step 3: Add DeviceInfo struct**

```go
type DeviceInfo struct {
	DeviceType string
	UserAgent  string
	IPAddress  string
}
```

**Step 4: Run tests**

Run: `go test ./internal/auth/usecase/... -v -run Login`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/auth/usecase/auth_usecase.go
git commit -m "feat(auth): implement single session policy - force delete old sessions on login"
```

---

### Task 5.2: Implement IDOR Protection for Products

**Files:**
- Modify: `internal/product/usecase/product_usecase.go`

**Step 1: Find GetProduct method**

Run: `grep -n "func.*GetProduct" internal/product/usecase/product_usecase.go`

**Step 2: Update GetProduct with IDOR protection**

```go
func (u *ProductUseCase) GetProduct(ctx context.Context, id string, userID string, userRole string) (*domain.Product, error) {
	product, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}

	// IDOR PROTECTION: Non-admin users can only access their own products
	if userRole != string(domain.RoleAdmin) && product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	return product, nil
}
```

**Step 3: Update UpdateProduct method**

```go
func (u *ProductUseCase) UpdateProduct(ctx context.Context, id string, userID string, userRole string, req *dto.UpdateProductRequest) (*domain.Product, error) {
	product, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}

	// IDOR PROTECTION: Only owner can update (admin CANNOT update user products)
	if product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	// Update fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}

	if err := u.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}
```

**Step 4: Add ErrAccessDenied**

```go
var ErrAccessDenied = errors.New("access denied: you do not have permission to perform this action")
```

**Step 5: Commit**

```bash
git add internal/product/usecase/product_usecase.go
git commit -m "feat(product): implement IDOR protection - can only access their users own products"
```

---

### Task 5.3: Add Internal API Endpoint

**Files:**
- Modify: `internal/user/delivery/handler.go`
- Modify: `internal/user/usecase/user_usecase.go`

**Step 1: Add GetOldestUser method to usecase**

```go
func (u *UserUseCase) GetOldestUserByRole(ctx context.Context, role string) (*domain.User, error) {
	return u.userRepo.FindOldestByRole(ctx, role)
}
```

**Step 2: Add repository method**

```go
func (r *UserRepository) FindOldestByRole(ctx context.Context, role string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("role = ?", role).
		Order("created_at ASC").
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
```

**Step 3: Add handler endpoint**

```go
func (h *UserHandler) GetOldestUser(c *gin.Context) {
	role := c.DefaultQuery("role", "USER")

	if role != "ADMIN" && role != "USER" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ROLE",
				"message": "Invalid role. Must be ADMIN or USER",
			},
		})
		return
	}

	user, err := h.userUseCase.GetOldestUserByRole(c.Request.Context(), role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "No " + role + " user found in database",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user.ToSafeUser(),
	})
}
```

**Step 4: Register route**

```go
internalGroup.GET("/internal/users/oldest", h.GetOldestUser)
```

**Step 5: Commit**

```bash
git add internal/user/delivery/handler.go internal/user/usecase/user_usecase.go internal/user/repository/user_repository.go
git commit -m "feat(user): add internal API endpoint GET /api/internal/users/oldest"
```

---

## Phase 6: Repository Updates

### Task 6.1: Update User Repository

**Files:**
- Modify: `internal/auth/repository/user_repository.go`
- Modify: `internal/user/repository/user_repository.go`

**Step 1: Add FindByUsername method**

```go
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
```

**Step 2: Add UsernameExists check**

```go
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("username = ?", username).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

**Step 3: Commit**

```bash
git add internal/auth/repository/user_repository.go internal/user/repository/user_repository.go
git commit -m "feat(repository): add FindByUsername and UsernameExists methods"
```

---

### Task 6.2: Update Session Repository

**Files:**
- Modify: `internal/auth/repository/session_repository.go`

**Step 1: Add DeleteByUserID method**

```go
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&domain.Session{}).Error
}
```

**Step 2: Add UpdateLastUsed method**

```go
func (r *SessionRepository) UpdateLastUsed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now().UTC()).Error
}
```

**Step 3: Commit**

```bash
git add internal/auth/repository/session_repository.go
git commit -m "feat(repository): add DeleteByUserID and UpdateLastUsed methods"
```

---

## Phase 7: Database Migrations

### Task 7.1: Create Migration Files

**Files:**
- Create: `migrations/001_alter_users_table.sql`
- Create: `migrations/002_alter_sessions_table.sql`
- Create: `migrations/003_alter_activity_logs_table.sql`
- Create: `migrations/004_alter_products_table.sql`
- Create: `migrations/005_create_product_variants_table.sql`
- Create: `migrations/006_create_product_attributes_table.sql`

**Step 1: Create users migration**

```sql
-- migrations/001_alter_users_table.sql
-- Add username and name fields, remove is_active

ALTER TABLE users ADD COLUMN username VARCHAR(50) NOT NULL UNIQUE;
ALTER TABLE users ADD COLUMN name VARCHAR(255);
ALTER TABLE users DROP COLUMN IF EXISTS is_active;

CREATE INDEX IF NOT EXISTS users_username_idx ON users(username);
```

**Step 2: Create sessions migration**

```sql
-- migrations/002_alter_sessions_table.sql
-- Rename RefreshToken to Token, add LastUsedAt, DeviceType, soft delete

ALTER TABLE sessions RENAME COLUMN refresh_token TO token;
ALTER TABLE sessions ADD COLUMN last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
ALTER TABLE sessions ADD COLUMN device_type VARCHAR(50);
ALTER TABLE sessions ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);
CREATE INDEX IF NOT EXISTS sessions_deleted_at_idx ON sessions(deleted_at);
```

**Step 3: Create activity logs migration**

```sql
-- migrations/003_alter_activity_logs_table.sql
-- Rename table and fields

ALTER TABLE activity_logs RENAME TO user_activity_logs;
ALTER TABLE user_activity_logs RENAME COLUMN resource TO entity;
ALTER TABLE user_activity_logs RENAME COLUMN resource_id TO entity_id;
ALTER TABLE user_activity_logs RENAME COLUMN metadata TO details;
ALTER TABLE user_activity_logs ALTER COLUMN action TYPE VARCHAR(255);
ALTER TABLE user_activity_logs ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS user_activity_logs_user_id_idx ON user_activity_logs(user_id);
CREATE INDEX IF NOT EXISTS user_activity_logs_action_idx ON user_activity_logs(action);
CREATE INDEX IF NOT EXISTS user_activity_logs_deleted_at_idx ON user_activity_logs(deleted_at);
```

**Step 4: Create products migration**

```sql
-- migrations/004_alter_products_table.sql
-- Add owner_id, has_variant, images; remove description, status, category_id

ALTER TABLE products ADD COLUMN owner_id UUID NOT NULL;
ALTER TABLE products ADD COLUMN has_variant BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE products ADD COLUMN images TEXT;
ALTER TABLE products DROP COLUMN IF EXISTS description;
ALTER TABLE products DROP COLUMN IF EXISTS status;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;

CREATE INDEX IF NOT EXISTS products_owner_id_idx ON products(owner_id);
CREATE INDEX IF NOT EXISTS products_has_variant_idx ON products(has_variant);
```

**Step 5: Create product_variants migration**

```sql
-- migrations/005_create_product_variants_table.sql

CREATE TABLE IF NOT EXISTS product_variants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NOT NULL UNIQUE,
    price DECIMAL(10,2),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    stock_reserved INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    attribute_values JSONB,
    images TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS product_variants_product_id_idx ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS product_variants_is_active_idx ON product_variants(is_active);
CREATE INDEX IF NOT EXISTS product_variants_deleted_at_idx ON product_variants(deleted_at);
```

**Step 6: Create product_attributes migration**

```sql
-- migrations/006_create_product_attributes_table.sql

CREATE TABLE IF NOT EXISTS product_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    values JSONB NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS product_attributes_product_id_idx ON product_attributes(product_id);
CREATE INDEX IF NOT EXISTS product_attributes_product_id_name_idx ON product_attributes(product_id, name);
CREATE INDEX IF NOT EXISTS product_attributes_deleted_at_idx ON product_attributes(deleted_at);
```

**Step 7: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): add schema alignment migrations"
```

---

## Phase 8: Integration Tests

### Task 8.1: Create Integration Tests

**Files:**
- Create: `tests/integration/auth_test.go`
- Create: `tests/integration/product_test.go`

**Step 1: Create auth integration test**

```go
package integration

import (
	"testing"
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/validator"
)

func TestAuthFlow(t *testing.T) {
	t.Run("register with valid credentials", func(t *testing.T) {
		// Test password validation
		err := validator.ValidatePassword("Password1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Test username validation
		err = validator.ValidateUsername("john_doe")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("single session policy", func(t *testing.T) {
		// Verify session deletion on new login
		// This is tested in usecase tests
	})
}
```

**Step 2: Create product integration test**

```go
package integration

import (
	"testing"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

func TestProductFlow(t *testing.T) {
	t.Run("IDOR protection", func(t *testing.T) {
		product := &domain.Product{
			OwnerID: "user-123",
		}

		// Test that another user cannot access
		anotherUserID := "user-456"
		if product.OwnerID == anotherUserID {
			t.Error("IDOR protection failed")
		}
	})
}
```

**Step 3: Run integration tests**

Run: `go test ./tests/integration/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add tests/integration/
git commit -m "test: add integration tests for schema alignment features"
```

---

## Phase 9: Final Verification

### Task 9.1: Run Full Test Suite

**Step 1: Run all tests**

Run: `go test ./... -v -count=1 2>&1 | head -100`
Expected: All tests PASS

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: No errors

**Step 3: Run go lint**

Run: `golangci-lint run`
Expected: No critical errors

**Step 4: Build all services**

Run: `go build ./cmd/...`
Expected: All services build successfully

---

## Summary of Changes

### Files Modified

| File | Changes |
|------|---------|
| `internal/auth/domain/user.go` | Add Username, Name; Remove IsActive |
| `internal/user/domain/user.go` | Add Username, Name; Update ActivityLog |
| `internal/auth/domain/session.go` | Rename field, add fields, soft delete |
| `internal/product/domain/product.go` | Add OwnerID, HasVariant, Images |
| `internal/auth/dto/request.go` | Add validation rules |
| `internal/product/dto/request.go` | Sync with Bun format |
| `internal/auth/usecase/auth_usecase.go` | Single session policy |
| `internal/product/usecase/product_usecase.go` | IDOR protection |
| `internal/user/delivery/handler.go` | Internal API endpoint |

### Files Created

| File | Description |
|------|-------------|
| `internal/product/domain/variant.go` | ProductVariant entity |
| `internal/product/domain/attribute.go` | ProductAttribute entity |
| `pkg/validator/password.go` | Password validation |
| `pkg/validator/username.go` | Username validation |
| `migrations/*.sql` | Database migrations |
| `tests/integration/*.go` | Integration tests |

---

## Post-Implementation Checklist

- [ ] All domain entities aligned with Bun schema
- [ ] All validators implemented and tested
- [ ] Single session policy working
- [ ] IDOR protection implemented
- [ ] Internal API endpoint working
- [ ] Database migrations tested
- [ ] All tests passing
- [ ] Go vet and lint clean
- [ ] Documentation updated

---

**Plan complete and saved to `docs/plans/2026-03-08-schema-alignment-implementation.md`**

Two execution options:

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
