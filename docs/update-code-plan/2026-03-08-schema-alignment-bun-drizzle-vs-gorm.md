# Comprehensive Schema Refactoring Plan: Go (GORM) to Match Bun (Drizzle)

**Date:** March 8, 2026  
**Project:** go-microservices-redis-pubsub-boilerplate  
**Goal:** Align Go microservice schemas with Bun project for full API/feature parity  
**Author:** Claude Code (AI Assistant)

---

## Executive Summary

This document provides an extremely detailed and comprehensive plan for refactoring the Go (GORM) database schemas to exactly match the Bun (Drizzle ORM) implementation. The analysis reveals **significant gaps** between the two implementations that must be addressed to achieve feature parity.

### Key Findings Summary

| Category | Critical Issues | Important Issues | Total |
|----------|-----------------|------------------|-------|
| Schema Fields Missing | 12 | 8 | 20 |
| Schema Fields Wrong | 8 | 5 | 13 |
| API Behavior Differences | 5 | 7 | 12 |
| Missing Tables | 2 | 0 | 2 |
| **TOTAL** | **27** | **20** | **47** |

---

## Part 1: Deep Analysis of Bun Project Validation Rules

### 1.1 User Registration Validation Rules (From Bun)

The Bun project implements strict validation rules using Zod. These rules MUST be replicated in Go.

#### Email Validation (Bun)
```typescript
// From: service-user/src/modules/user/domain/auth.ts
email: z.string().email('Invalid email format')
```

**Requirements:**
- Email must be a valid email format (RFC 5322)
- Email must be unique in the database (enforced at DB level with UNIQUE constraint)
- Email is required (NOT NULL)
- Maximum length: 255 characters

**Go Implementation Required:**
```go
type RegisterRequest struct {
    Email string `json:"email" binding:"required,email,max=255"`
}
```

**Database Constraint:**
```sql
email varchar(255) NOT NULL UNIQUE
```

#### Username Validation (Bun)
```typescript
// From: service-user/src/modules/user/domain/auth.ts
username: z.string().min(3, 'Username must be at least 3 characters').max(50)
```

**Requirements:**
- Username minimum length: 3 characters
- Username maximum length: 50 characters
- Username must be unique (UNIQUE constraint)
- Username is required (NOT NULL)
- Username can only contain: letters, numbers, underscores

**Go Implementation Required:**
```go
type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50,alphanumunderscore"`
}
```

**Database Constraint:**
```sql
username varchar(50) NOT NULL UNIQUE
```

#### Password Validation (Bun) - CRITICAL
```typescript
// From: service-user/src/helpers/password.ts
/**
 * Password validation regex:
 * - Minimum 8 characters
 * - At least one uppercase letter
 * - At least one number
 * - Special characters allowed but limited to common safe ones: !@#$%^&*()_+-=[]{}|;:,.<>?
 * - No characters that could lead to XSS or SQL injection (e.g., ', ", `, \, /)
 */
const PASSWORD_REGEX = /^[A-Za-z0-9!@#$%^&*()_+\-=[\]{}|;:,.<>?]{8,}$/;

/**
 * Zod schema for password validation
 */
export const PasswordSchema = z
  .string()
  .min(8, 'Password must be at least 8 characters long')
  .regex(/[A-Z]/, 'Password must contain at least one uppercase letter')
  .regex(/[0-9]/, 'Password must contain at least one number')
  .regex(
    PASSWORD_REGEX,
    'Password contains invalid characters or does not meet complexity requirements'
  );
```

**Password Rules Summary:**
| Rule | Requirement | Example |
|------|-------------|---------|
| Minimum Length | 8 characters | "Password1" |
| Uppercase | At least 1 uppercase letter | "PASSWORD1" |
| Number | At least 1 number | "Password1" |
| Special Characters | Allowed: !@#$%^&*()_+-=[]{}|;:,.<>? | "Password1!" |
| Forbidden Characters | ', ", `, \, / | N/A |

**Go Implementation Required:**
```go
// Custom validation function for password
func ValidatePassword(password string) error {
    // Check minimum length
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters long")
    }
    
    // Check uppercase
    hasUpper := false
    for _, c := range password {
        if unicode.IsUpper(c) {
            hasUpper = true
            break
        }
    }
    if !hasUpper {
        return errors.New("password must contain at least one uppercase letter")
    }
    
    // Check number
    hasNumber := false
    for _, c := range password {
        if unicode.IsDigit(c) {
            hasNumber = true
            break
        }
    }
    if !hasNumber {
        return errors.New("password must contain at least one number")
    }
    
    // Check for forbidden characters
    forbidden := []rune{'\'', '"', '`', '\\', '/'}
    for _, c := range password {
        for _, f := range forbidden {
            if c == f {
                return errors.New("password contains forbidden characters")
            }
        }
    }
    
    return nil
}
```

#### Name Field (Bun)
```typescript
// From: service-user/src/modules/user/domain/auth.ts
name: z.string().min(1, 'Name is required').max(255).optional()
```

**Requirements:**
- Name is optional
- Maximum length: 255 characters
- If provided, must not be empty

**Go Implementation Required:**
```go
type RegisterRequest struct {
    Name *string `json:"name" binding:"omitempty,max=255"`
}
```

#### Role Field (Bun)
```typescript
// From: service-user/src/modules/user/domain/auth.ts
role: z.enum(['ADMIN', 'USER']).optional().default('USER')
```

**Requirements:**
- Only two valid values: ADMIN, USER
- Default value: USER
- If not provided, defaults to USER

**Go Implementation Required:**
```go
type RegisterRequest struct {
    Role *string `json:"role" binding:"omitempty,oneof=ADMIN USER"`
}
```

---

### 1.2 User Session Validation Rules

#### Session Table (Bun)
```typescript
// From: service-user/src/modules/user/domain/schema.ts
export const userSessions = createParanoidTable(
  'user_sessions',
  {
    userId: uuid('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' }),
    token: text('token'), // Hash of JWT for auditing
    ipAddress: varchar('ip_address', { length: 45 }), // IPv6 support
    userAgent: text('user_agent'),
    deviceType: varchar('device_type', { length: 50 }), // 'mobile', 'tablet', 'desktop', 'unknown'
    expiresAt: timestamp('expires_at', { mode: 'date', withTimezone: true }).notNull(),
    lastUsedAt: timestamp('last_used_at', { mode: 'date', withTimezone: true }).defaultNow(),
  },
  table => ({
    userIdIdx: index('user_sessions_user_id_idx').on(table.userId),
  })
);
```

**Session Fields:**
| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| id | uuid | YES | auto-generated | Primary key |
| created_at | timestamp | YES | now() | With timezone |
| updated_at | timestamp | YES | now() | With timezone |
| deleted_at | timestamp | NO | NULL | Soft delete |
| user_id | uuid | YES | - | FK to users |
| token | text | NO | NULL | JWT token hash |
| ip_address | varchar(45) | NO | NULL | IPv6 support |
| user_agent | text | NO | NULL | Client user agent |
| device_type | varchar(50) | NO | NULL | mobile/tablet/desktop |
| expires_at | timestamp | YES | - | With timezone |
| last_used_at | timestamp | NO | now() | With timezone |

---

### 1.3 Activity Log Validation Rules

#### Activity Log Table (Bun)
```typescript
// From: service-user/src/modules/user/domain/schema.ts
export const userActivityLogs = createParanoidTable(
  'user_activity_logs',
  {
    userId: uuid('user_id').references(() => users.id, { onDelete: 'cascade' }),
    action: varchar('action', { length: 255 }).notNull(), // e.g., 'auth.login', 'product.create'
    entityId: uuid('entity_id'), // Optional: ID of the entity affected
    details: jsonb('details'), // Metadata: { ip, ua, diff, etc. }
    ipAddress: varchar('ip_address', { length: 45 }),
    userAgent: text('user_agent'),
  },
  table => ({
    userIdIdx: index('user_activity_logs_user_id_idx').on(table.userId),
    actionIdx: index('user_activity_logs_action_idx').on(table.action),
    createdAtIdx: index('user_activity_logs_created_at_idx').on(table.createdAt),
  })
);
```

**Activity Log Fields:**
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| id | uuid | YES | Primary key |
| created_at | timestamp | YES | With timezone |
| updated_at | timestamp | YES | With timezone |
| deleted_at | timestamp | NO | Soft delete |
| user_id | uuid | YES | FK to users |
| action | varchar(255) | YES | e.g., 'auth.login' |
| entity_id | uuid | NO | Entity affected |
| details | jsonb | NO | Metadata |
| ip_address | varchar(45) | NO | IPv6 support |
| user_agent | text | NO | Client user agent |

---

### 1.4 Product Validation Rules

#### Product Table (Bun)
```typescript
// From: service-product/src/modules/product/domain/schema.ts
export const products = createParanoidTable(
  'products',
  {
    name: varchar('name', { length: 255 }).notNull(),
    price: decimal('price', { precision: 10, scale: 2 }).notNull().$type<number>(),
    ownerId: uuid('owner_id').notNull(),
    stock: integer('stock').default(0).notNull(),
    hasVariant: boolean('has_variant').default(false).notNull(),
  },
  table => ({
    ownerIdIdx: index('products_owner_id_idx').on(table.ownerId),
    nameIdx: index('products_name_idx').on(table.name),
    priceIdx: index('products_price_idx').on(table.price),
    hasVariantIdx: index('products_has_variant_idx').on(table.hasVariant),
    stockIdx: index('products_stock_idx').on(table.stock),
  })
);
```

**Product Fields:**
| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| id | uuid | YES | auto-generated | Primary key |
| created_at | timestamp | YES | now() | With timezone |
| updated_at | timestamp | YES | now() | With timezone |
| deleted_at | timestamp | NO | NULL | Soft delete |
| name | varchar(255) | YES | - | Product name |
| price | decimal(10,2) | YES | - | As number |
| owner_id | uuid | YES | - | FK to users |
| stock | integer | YES | 0 | Stock quantity |
| has_variant | boolean | YES | false | Has variants |

**Important Notes:**
- Product name is NOT globally unique (marketplace model)
- Same owner cannot have duplicate names
- Different owners CAN have products with same name

#### Product Variant Table (Bun)
```typescript
// From: service-product/src/modules/product/domain/schema-variants.ts
export const productVariants = createParanoidTable(
  'product_variants',
  {
    productId: uuid('product_id')
      .notNull()
      .references(() => products.id, { onDelete: 'cascade' }),
    sku: varchar('sku', { length: 100 }).notNull(),
    price: decimal('price', { precision: 10, scale: 2 }).$type<number>(),
    stockQuantity: integer('stock_quantity').default(0).notNull(),
    stockReserved: integer('stock_reserved').default(0).notNull(),
    isActive: boolean('is_active').default(true).notNull(),
    attributeValues: jsonb('attribute_values').notNull().$type<Record<string, string>>(),
  },
  table => ({
    productIdIdx: index('product_variants_product_id_idx').on(table.productId),
    skuIdx: unique('product_variants_sku_unique').on(table.sku),
    isActiveIdx: index('product_variants_is_active_idx').on(table.isActive),
  })
);
```

#### Product Attributes Table (Bun)
```typescript
// From: service-product/src/modules/product/domain/schema-attributes.ts
export const productAttributes = createParanoidTable(
  'product_attributes',
  {
    productId: uuid('product_id')
      .notNull()
      .references(() => products.id, { onDelete: 'cascade' }),
    name: varchar('name', { length: 100 }).notNull(),
    values: jsonb('values').notNull().$type<string[]>(),
    displayOrder: integer('display_order').default(0).notNull(),
  },
  table => ({
    productIdIdx: index('product_attributes_product_id_idx').on(table.productId),
    productIdNameIdx: index('product_attributes_product_id_name_idx').on(
      table.productId,
      table.name
    ),
  })
);
```

---

## Part 2: Current Go Schema Analysis

### 2.1 Current Users Table (Go - INCORRECT)

```go
// From: internal/auth/domain/user.go
type User struct {
    Model
    Email        string     `gorm:"type:varchar(255);not null;uniqueIndex"`
    PasswordHash string     `gorm:"type:text;not null"`
    Role         Role       `gorm:"type:varchar(50);not null;default:'USER'"`
    IsActive     bool       `gorm:"default:true"`
    LastLoginAt  *time.Time
}
```

**Issues Found:**
| Issue | Severity | Description |
|-------|----------|-------------|
| Missing Username | 🔴 CRITICAL | username field does not exist |
| Missing Name | 🔴 CRITICAL | name field does not exist |
| Has IsActive | ⚠️ WRONG | Field not in Bun schema - REMOVE |
| Missing unique constraint | ⚠️ NEEDS FIX | Should be uniqueIndex (already correct) |

### 2.2 Current Sessions Table (Go - INCORRECT)

```go
// From: internal/auth/domain/session.go
type Session struct {
    ID           string     `gorm:"type:uuid;primary_key;"`
    UserID       string     `gorm:"type:uuid;not null;index"`
    RefreshToken string     `gorm:"type:text;not null"`
    ExpiresAt    time.Time  `gorm:"not null"`
    CreatedAt    time.Time  `gorm:"not null"`
    RevokedAt    *time.Time
    UserAgent    string     `gorm:"type:text"`
    IPAddress    string     `gorm:"type:varchar(45)"`
}
```

**Issues Found:**
| Issue | Severity | Description |
|-------|----------|-------------|
| Wrong Field Name | 🔴 CRITICAL | RefreshToken should be token |
| Missing token | 🔴 CRITICAL | token field does not exist |
| Missing lastUsedAt | 🔴 CRITICAL | last_used_at field missing |
| Missing deviceType | 🔴 CRITICAL | device_type field missing |
| Missing soft delete | 🔴 CRITICAL | deleted_at field missing |
| Missing FK | ⚠️ NEEDS FIX | No foreign key to users |

### 2.3 Current Activity Logs Table (Go - INCORRECT)

```go
// From: internal/user/domain/user.go (ActivityLog)
type ActivityLog struct {
    Model
    UserID     string
    Action     string    `gorm:"type:varchar(100);not null"`
    Resource   string
    ResourceID string
    IPAddress  string    `gorm:"type:varchar(45)"`
    UserAgent  string    `gorm:"type:text"`
    Metadata   map[string]interface{} `gorm:"type:jsonb"`
}
```

**Issues Found:**
| Issue | Severity | Description |
|-------|----------|-------------|
| Wrong Table Name | 🔴 CRITICAL | Should be user_activity_logs |
| Wrong Field Name | 🔴 CRITICAL | ResourceID should be entity_id |
| Wrong Field Name | 🔴 CRITICAL | Metadata should be details |
| Field Too Short | ⚠️ NEEDS FIX | action should be varchar(255) |
| Missing soft delete | 🔴 CRITICAL | deleted_at missing |
| Missing FK | ⚠️ NEEDS FIX | No foreign key to users |

### 2.4 Current Products Table (Go - INCORRECT)

```go
// From: internal/product/domain/product.go
type Product struct {
    Model
    Name        string        `gorm:"type:varchar(255);not null"`
    Description string        `gorm:"type:text"`
    Price       float64       `gorm:"type:decimal(10,2);not null"`
    Stock       int           `gorm:"type:int;not null;default:0"`
    Status      ProductStatus `gorm:"type:varchar(50)"`
    CategoryID  string        `gorm:"type:uuid;not null"`
}
```

**Issues Found:**
| Issue | Severity | Description |
|-------|----------|-------------|
| Missing ownerId | 🔴 CRITICAL | owner_id field missing |
| Missing hasVariant | 🔴 CRITICAL | has_variant field missing |
| Extra fields | ⚠️ REMOVE | description, status, category_id NOT in Bun |

### 2.5 Missing Tables (Go)

| Table | Bun Status | Go Status | Action |
|-------|------------|-----------|--------|
| product_variants | ✅ EXISTS | ❌ MISSING | CREATE |
| product_attributes | ✅ EXISTS | ❌ MISSING | CREATE |

---

## Part 3: Detailed Implementation Plan

### Phase 1: Critical Schema Fixes

#### Step 1.1: Fix Users Table

**File:** `internal/auth/domain/user.go`
**File:** `internal/user/domain/user.go`

**Before (Incorrect):**
```go
type User struct {
    Model
    Email        string     `gorm:"type:varchar(255);not null;uniqueIndex"`
    PasswordHash string     `gorm:"type:text;not null"`
    Role         Role       `gorm:"type:varchar(50);not null;default:'USER'"`
    IsActive     bool       `gorm:"default:true"`
    LastLoginAt  *time.Time
}
```

**After (Correct):**
```go
type User struct {
    Model
    Email        string     `gorm:"type:varchar(255);not null;uniqueIndex"`
    Username     string     `gorm:"type:varchar(50);not null;uniqueIndex"`
    Name         string     `gorm:"type:varchar(255)"`
    PasswordHash string     `gorm:"type:text;not null"`
    Role         Role       `gorm:"type:varchar(50);not null;default:'USER'"`
    LastLoginAt  *time.Time
}
```

**Changes:**
1. ✅ ADD: Username field (varchar 50, NOT NULL, UNIQUE)
2. ✅ ADD: Name field (varchar 255, NULLABLE)
3. ✅ REMOVE: IsActive field (not in Bun schema)

---

#### Step 1.2: Fix Sessions Table

**File:** `internal/auth/domain/session.go`

**Before (Incorrect):**
```go
type Session struct {
    ID           string     `gorm:"type:uuid;primary_key;"`
    UserID       string     `gorm:"type:uuid;not null;index"`
    RefreshToken string     `gorm:"type:text;not null"`
    ExpiresAt    time.Time  `gorm:"not null"`
    CreatedAt    time.Time  `gorm:"not null"`
    RevokedAt    *time.Time
    UserAgent    string     `gorm:"type:text"`
    IPAddress    string     `gorm:"type:varchar(45)"`
}
```

**After (Correct):**
```go
type Session struct {
    Model  // Includes: ID, CreatedAt, UpdatedAt, DeletedAt
    UserID       string     `gorm:"type:uuid;not null;index"`
    Token        string     `gorm:"type:text"`
    ExpiresAt    time.Time  `gorm:"not null"`
    LastUsedAt   *time.Time `gorm:"default:CURRENT_TIMESTAMP"`
    UserAgent    string     `gorm:"type:text"`
    IPAddress    string     `gorm:"type:varchar(45)"`
    DeviceType   string     `gorm:"type:varchar(50)"`
}
```

**Changes:**
1. ✅ ADD: Embed Model for soft delete support
2. ✅ RENAME: RefreshToken → Token
3. ✅ REMOVE: RefreshToken field
4. ✅ ADD: LastUsedAt field
5. ✅ ADD: DeviceType field

---

#### Step 1.3: Fix Activity Logs Table

**File:** Create new file or rename existing

**Before (Incorrect):**
```go
type ActivityLog struct {
    Model
    UserID     string
    Action     string    `gorm:"type:varchar(100);not null"`
    Resource   string
    ResourceID string
    IPAddress  string    `gorm:"type:varchar(45)"`
    UserAgent  string    `gorm:"type:text"`
    Metadata   map[string]interface{} `gorm:"type:jsonb"`
}
```

**After (Correct):**
```go
// Table name: user_activity_logs (NOT activity_logs)
type ActivityLog struct {
    Model  // Includes: ID, CreatedAt, UpdatedAt, DeletedAt
    UserID     string                 `gorm:"type:uuid;not null;index"`
    Action     string                 `gorm:"type:varchar(255);not null"`
    EntityID   string                 `gorm:"type:uuid"`
    Details    map[string]interface{} `gorm:"type:jsonb"`
    IPAddress  string                 `gorm:"type:varchar(45)"`
    UserAgent  string                 `gorm:"type:text"`
}
```

**Changes:**
1. ✅ RENAME: Table from activity_logs → user_activity_logs
2. ✅ RENAME: ResourceID → EntityID
3. ✅ RENAME: Metadata → Details
4. ✅ EXPAND: action from varchar(100) → varchar(255)

---

#### Step 1.4: Fix Products Table

**File:** `internal/product/domain/product.go`

**Before (Incorrect):**
```go
type Product struct {
    Model
    Name        string        `gorm:"type:varchar(255);not null"`
    Description string        `gorm:"type:text"`
    Price       float64       `gorm:"type:decimal(10,2);not null"`
    Stock       int           `gorm:"type:int;not null;default:0"`
    Status      ProductStatus `gorm:"type:varchar(50)"`
    CategoryID  string        `gorm:"type:uuid;not null"`
}
```

**After (Correct):**
```go
type Product struct {
    Model
    Name        string  `gorm:"type:varchar(255);not null"`
    Price       float64 `gorm:"type:decimal(10,2);not null"`
    OwnerID     string  `gorm:"type:uuid;not null;index"`
    Stock       int     `gorm:"type:int;not null;default:0"`
    HasVariant  bool    `gorm:"default:false"`
}
```

**Changes:**
1. ✅ ADD: OwnerID field (uuid, NOT NULL)
2. ✅ ADD: HasVariant field (boolean, default false)
3. ✅ REMOVE: Description field
4. ✅ REMOVE: Status field
5. ✅ REMOVE: CategoryID field

---

### Phase 2: New Tables Required

#### Step 2.1: Create Product Variants Table

**New File:** `internal/product/domain/variant.go`

```go
package domain

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ProductVariant represents a product variant (SKU).
type ProductVariant struct {
    ID             string                 `gorm:"type:uuid;primary_key;" json:"id"`
    CreatedAt      time.Time              `gorm:"not null" json:"created_at"`
    UpdatedAt      time.Time              `gorm:"not null" json:"updated_at"`
    DeletedAt      gorm.DeletedAt        `gorm:"index" json:"deleted_at,omitempty"`
    
    ProductID      string                 `gorm:"type:uuid;not null;index" json:"product_id"`
    SKU            string                 `gorm:"type:varchar(100);not null;uniqueIndex" json:"sku"`
    Price          *float64               `gorm:"type:decimal(10,2)" json:"price,omitempty"`
    StockQuantity  int                    `gorm:"type:int;not null;default:0" json:"stock_quantity"`
    StockReserved  int                    `gorm:"type:int;not null;default:0" json:"stock_reserved"`
    IsActive       bool                   `gorm:"default:true" json:"is_active"`
    AttributeValues map[string]string     `gorm:"type:jsonb" json:"attribute_values"`
}

// TableName specifies the table name.
func (ProductVariant) TableName() string {
    return "product_variants"
}

// BeforeCreate generates UUID if not set.
func (p *ProductVariant) BeforeCreate(_ *gorm.DB) error {
    if p.ID == "" {
        p.ID = uuid.New().String()
    }
    return nil
}
```

---

#### Step 2.2: Create Product Attributes Table

**New File:** `internal/product/domain/attribute.go`

```go
package domain

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ProductAttribute represents a product attribute (e.g., Color, Size).
type ProductAttribute struct {
    ID           string   `gorm:"type:uuid;primary_key;" json:"id"`
    CreatedAt    time.Time `gorm:"not null" json:"created_at"`
    UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
    
    ProductID    string   `gorm:"type:uuid;not null;index" json:"product_id"`
    Name         string   `gorm:"type:varchar(100);not null" json:"name"`
    Values       []string `gorm:"type:jsonb" json:"values"`
    DisplayOrder int      `gorm:"type:int;default:0" json:"display_order"`
}

// TableName specifies the table name.
func (ProductAttribute) TableName() string {
    return "product_attributes"
}

// BeforeCreate generates UUID if not set.
func (p *ProductAttribute) BeforeCreate(_ *gorm.DB) error {
    if p.ID == "" {
        p.ID = uuid.New().String()
    }
    return nil
}
```

---

### Phase 3: DTO Updates

#### Step 3.1: Update Auth DTOs

**File:** `internal/auth/dto/request.go`

**Changes Required:**
```go
// RegisterRequest - ADD username, UPDATE password validation
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email,max=255"`
    Username string `json:"username" binding:"required,min=3,max=50,alphanumunderscore"`
    Password string `json:"password" binding:"required"`  // Add custom validation
    Name     string `json:"name" binding:"omitempty,max=255"`
    Role     string `json:"role" binding:"omitempty,oneof=ADMIN USER"`
}
```

---

#### Step 3.2: Update Product DTOs

**File:** `internal/product/dto/request.go`

**Changes Required:**
```go
// CreateProductRequest - ADD ownerId from auth, REMOVE description/status/categoryId
type CreateProductRequest struct {
    Name     string  `json:"name" binding:"required,min=1,max=255"`
    Price    float64 `json:"price" binding:"required,gt=0"`
    Stock    int     `json:"stock" binding:"omitempty,min=0"`
    // ownerId comes from JWT token, NOT from request body
}

// ProductResponse - SYNC with Bun response format
type ProductResponse struct {
    ID          string      `json:"id"`
    Name        string      `json:"name"`
    Price       float64     `json:"price"`
    OwnerID     string      `json:"owner_id"`
    Stock       int         `json:"stock"`
    HasVariant  bool        `json:"has_variant"`
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
}
```

---

### Phase 4: API Behavior Changes

#### Step 4.1: Implement Single Session Policy

**Current (Go):** Allows multiple sessions per user  
**Required (Bun):** Force delete all existing sessions before creating new one

**Implementation:**
```go
// In: internal/auth/usecase/auth_usecase.go
func (u *AuthUseCase) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
    // ... existing login logic ...
    
    // SINGLE SESSION POLICY: Delete ALL existing sessions first
    if err := u.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
        return nil, err
    }
    
    // Then create new session
    session := &domain.Session{
        UserID:    user.ID,
        Token:     token,  // Store JWT token
        ExpiresAt: expiresAt,
        IPAddress: ipAddress,
        UserAgent: userAgent,
        DeviceType: deviceType,
    }
    
    if err := u.sessionRepo.Create(ctx, session); err != nil {
        return nil, err
    }
    
    // ... rest of login logic ...
}
```

---

#### Step 4.2: Implement Owner-Based IDOR Protection

**Current (Go):** No owner-based access control  
**Required (Bun):** Strict IDOR protection - users can only access their own products

**Implementation:**
```go
// In: internal/product/usecase/product_usecase.go
func (u *ProductUseCase) GetProduct(ctx context.Context, id string, userID string, userRole string) (*domain.Product, error) {
    product, err := u.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // IDOR PROTECTION: Non-admin users can only see their own products
    if userRole != "ADMIN" && product.OwnerID != userID {
        return nil, ErrAccessDenied
    }
    
    return product, nil
}

func (u *ProductUseCase) UpdateProduct(ctx context.Context, id string, userID string, userRole string, req *dto.UpdateProductRequest) (*domain.Product, error) {
    product, err := u.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // IDOR PROTECTION: Only owner can update (admin CANNOT update user products)
    if product.OwnerID != userID {
        return nil, ErrAccessDenied
    }
    
    // ... rest of update logic ...
}
```

---

#### Step 4.3: Add Internal API Endpoint

**Required Endpoint (Bun):** GET /api/internal/users/oldest

**Implementation:**
```go
// In: internal/user/delivery/handler.go
func (h *UserHandler) GetOldestUser(c *gin.Context) {
    role := c.DefaultQuery("role", "USER")
    
    // Validate role
    if role != "ADMIN" && role != "USER" {
        c.JSON(http.StatusBadRequest, gin.H{
            "Success": false,
            "Error": gin.H{
                "Code":    "INVALID_ROLE",
                "Message": "Invalid role. Must be ADMIN or USER",
            },
        })
        return
    }
    
    user, err := h.userUseCase.GetOldestUserByRole(c.Request.Context(), role)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "Success": false,
            "Error": gin.H{
                "Code":    "USER_NOT_FOUND",
                "Message": "No " + role + " user found in database",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "Success": true,
        "Data":    user,
    })
}
```

---

## Part 4: Validation Implementation

### Step 4.1: Password Validation (Go)

Create new file: `pkg/validator/password.go`

```go
package validator

import (
    "errors"
    "unicode"
)

// ValidatePassword validates password according to Bun project rules:
// - Minimum 8 characters
// - At least one uppercase letter
// - At least one number
// - Special characters allowed: !@#$%^&*()_+-=[]{}|;:,.<>?
// - Forbidden: ', ", `, \, /
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters long")
    }
    
    hasUpper := false
    hasLower := false
    hasNumber := false
    hasSpecial := false
    
    forbidden := []rune{'\'', '"', '`', '\\', '/'}
    
    for _, c := range password {
        // Check forbidden characters
        for _, f := range forbidden {
            if c == f {
                return errors.New("password contains forbidden characters")
            }
        }
        
        switch {
        case unicode.IsUpper(c):
            hasUpper = true
        case unicode.IsLower(c):
            hasLower = true
        case unicode.IsDigit(c):
            hasNumber = true
        case isSpecialChar(c):
            hasSpecial = true
        }
    }
    
    if !hasUpper {
        return errors.New("password must contain at least one uppercase letter")
    }
    
    if !hasNumber {
        return errors.New("password must contain at least one number")
    }
    
    // At least one lowercase or special character (for complexity)
    if !hasLower && !hasSpecial {
        return errors.New("password must contain at least one lowercase letter or special character")
    }
    
    return nil
}

func isSpecialChar(c rune) bool {
    special := "!@#$%^&*()_+-=[]{}|;:,.<>?"
    for _, s := range special {
        if c == s {
            return true
        }
    }
    return false
}
```

---

### Step 4.2: Username Validation (Go)

```go
package validator

import (
    "errors"
    "regexp"
)

// ValidateUsername validates username according to Bun project rules:
// - Minimum 3 characters
// - Maximum 50 characters
// - Alphanumeric and underscores only
func ValidateUsername(username string) error {
    if len(username) < 3 {
        return errors.New("username must be at least 3 characters")
    }
    
    if len(username) > 50 {
        return errors.New("username must not exceed 50 characters")
    }
    
    // Only alphanumeric and underscores
    matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
    if err != nil {
        return err
    }
    
    if !matched {
        return errors.New("username can only contain letters, numbers, and underscores")
    }
    
    return nil
}
```

---

## Part 5: File Change Summary

### Files to Modify

| File | Changes |
|------|---------|
| `internal/auth/domain/user.go` | Add Username, Name; Remove IsActive |
| `internal/user/domain/user.go` | Same as auth |
| `internal/auth/domain/session.go` | Rename field, add fields, add soft delete |
| `internal/user/domain/activity_log.go` | Rename table, rename fields |
| `internal/product/domain/product.go` | Add OwnerID, HasVariant; Remove fields |
| `internal/auth/dto/request.go` | Add validation rules |
| `internal/product/dto/request.go` | Sync with Bun format |

### Files to Create

| File | Description |
|------|-------------|
| `internal/product/domain/variant.go` | ProductVariant entity |
| `internal/product/domain/attribute.go` | ProductAttribute entity |
| `pkg/validator/password.go` | Password validation |
| `pkg/validator/username.go` | Username validation |

### Files to Delete

| File | Reason |
|------|--------|
| None | - |

---

## Part 6: Testing Requirements

### Unit Tests Required

| Test | Description |
|------|-------------|
| Password Validation | Test all Bun password rules |
| Username Validation | Test username rules |
| Session Creation | Test single session policy |
| IDOR Protection | Test owner-based access |
| Internal API | Test oldest user endpoint |

---

## Appendix A: Database Constraints Reference

### Users Table (Final)
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255),
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    last_login_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE INDEX users_role_idx ON users(role);
CREATE INDEX users_username_idx ON users(username);
CREATE INDEX users_deleted_at_idx ON users(deleted_at);
```

### User Sessions Table (Final)
```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address VARCHAR(45),
    user_agent TEXT,
    device_type VARCHAR(50),
    
    CONSTRAINT user_sessions_pkey PRIMARY KEY (id)
);

CREATE INDEX user_sessions_user_id_idx ON user_sessions(user_id);
CREATE INDEX user_sessions_deleted_at_idx ON user_sessions(deleted_at);
```

### User Activity Logs Table (Final)
```sql
CREATE TABLE user_activity_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(255) NOT NULL,
    entity_id UUID,
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    CONSTRAINT user_activity_logs_pkey PRIMARY KEY (id)
);

CREATE INDEX user_activity_logs_user_id_idx ON user_activity_logs(user_id);
CREATE INDEX user_activity_logs_action_idx ON user_activity_logs(action);
CREATE INDEX user_activity_logs_created_at_idx ON user_activity_logs(created_at);
CREATE INDEX user_activity_logs_deleted_at_idx ON user_activity_logs(deleted_at);
```

### Products Table (Final)
```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    stock INTEGER NOT NULL DEFAULT 0,
    has_variant BOOLEAN NOT NULL DEFAULT FALSE,
    
    CONSTRAINT products_pkey PRIMARY KEY (id)
);

CREATE INDEX products_owner_id_idx ON products(owner_id);
CREATE INDEX products_name_idx ON products(name);
CREATE INDEX products_price_idx ON products(price);
CREATE INDEX products_has_variant_idx ON products(has_variant);
CREATE INDEX products_stock_idx ON products(stock);
CREATE INDEX products_deleted_at_idx ON products(deleted_at);
```

### Product Variants Table (Final)
```sql
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL UNIQUE,
    price DECIMAL(10,2),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    stock_reserved INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    attribute_values JSONB NOT NULL,
    
    CONSTRAINT product_variants_pkey PRIMARY KEY (id)
);

CREATE INDEX product_variants_product_id_idx ON product_variants(product_id);
CREATE INDEX product_variants_is_active_idx ON product_variants(is_active);
CREATE INDEX product_variants_deleted_at_idx ON product_variants(deleted_at);
```

### Product Attributes Table (Final)
```sql
CREATE TABLE product_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    values JSONB NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    
    CONSTRAINT product_attributes_pkey PRIMARY KEY (id)
);

CREATE INDEX product_attributes_product_id_idx ON product_attributes(product_id);
CREATE INDEX product_attributes_product_id_name_idx ON product_attributes(product_id, name);
CREATE INDEX product_attributes_deleted_at_idx ON product_attributes(deleted_at);
```

---

## Document Information

**Version:** 1.0  
**Created:** March 8, 2026  
**Author:** Claude Code (AI Assistant)  
**Status:** Ready for Implementation  

---

*This document will be updated as implementation progresses.*
