# API Response Alignment Plan: Go Microservices vs Bun-Hono

## Overview

This document outlines the plan to align API responses between the Go microservice project and the Bun-Hono reference project. The goal is to ensure consistent response structures across both implementations.

---

## Executive Summary

### Current State
- **Go Project**: 27 endpoints across auth, user, and product services
- **Bun-Hono Project**: 27 endpoints across auth, user, and product services

### Key Differences Identified
| Category | Go Project | Bun-Hono Project | Status |
|----------|------------|------------------|--------|
| Product Response | Basic fields + images | Price range + variants + attributes | ⚠️ Needs Update |
| Auth Response | access_token + refresh_token + expires_in | token only (no refresh) | ⚠️ Needs Update |
| User Response | Has profile embedded | No profile embedded | ⚠️ Needs Update |
| Pagination | HasNext, HasPrev | hasNextPage, hasPreviousPage | ⚠️ Needs Update |
| Activity Logs | Has endpoint | No endpoint | 🔄 Feature Gap |
| Product Variants | Not implemented | Fully implemented | 🔄 Feature Gap |

---

## Detailed API Comparison

### 1. AUTH SERVICE

#### 1.1 Login Response

**Bun-Hono (Expected):**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "johndoe",
      "name": "John Doe",
      "role": "USER"
    }
  },
  "meta": null
}
```

**Go Project (Current):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "name": "John Doe",
    "role": "USER",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "deleted_at": null,
    "last_login_at": null
  }
}
```

**Differences:**
| Field | Bun-Hono | Go | Action |
|-------|----------|-----|--------|
| Root structure | success/data/meta wrapper | Direct fields | Wrap in standardized response |
| token | `token` | `access_token` + `refresh_token` | Use single `token` |
| expires_in | Not in data | Present | Add if needed |
| user.created_at | Not included | Included | Remove from login |
| user.updated_at | Not included | Included | Remove from login |
| user.deleted_at | Not included | Included | Remove from login |
| user.last_login_at | Not included | Included | Remove from login |

#### 1.2 Register Response

**Bun-Hono:** Same structure as login

**Go Project:** Same as login (already consistent)

---

### 2. USER SERVICE

#### 2.1 GET /me Response

**Bun-Hono:**
```json
{
  "success": true,
  "message": "User fetched successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "name": "John Doe",
    "role": "USER",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z",
    "deletedAt": null
  },
  "meta": null
}
```

**Go Project (Current):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "username": "johndoe",
  "name": "John Doe",
  "role": "USER",
  "profile": {
    "id": "uuid",
    "first_name": "John",
    "last_name": "Doe",
    "full_name": "John Doe",
    "avatar": "http://...",
    "bio": "User bio"
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "deleted_at": null,
  "last_login_at": null
}
```

**Differences:**
| Field | Bun-Hono | Go | Action |
|-------|----------|-----|--------|
| Root structure | success/data/meta wrapper | Direct fields | Wrap in standardized response |
| profile | Not included | Included | Remove or make optional |
| createdAt | camelCase | snake_case | Use camelCase |
| updatedAt | camelCase | snake_case | Use camelCase |
| deletedAt | camelCase | snake_case | Use camelCase |

#### 2.2 GET /admin/users Response (List Users)

**Bun-Hono:**
```json
{
  "success": true,
  "message": "Users fetched successfully",
  "data": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "username": "johndoe",
      "name": "John Doe",
      "role": "USER",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z",
      "deletedAt": null
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 100,
    "totalPages": 10,
    "hasNextPage": true,
    "hasPreviousPage": false
  }
}
```

**Go Project (Current):**
```json
{
  "users": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "username": "johndoe",
      "name": "John Doe",
      "role": "USER",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "deleted_at": null,
      "last_login_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 100,
    "total_pages": 10,
    "has_next": true,
    "has_prev": false
  }
}
```

**Differences:**
| Field | Bun-Hono | Go | Action |
|-------|----------|-----|--------|
| Root structure | success/data/meta | users/pagination | Use standardized response |
| List key | `data` array | `users` | Use `data` |
| Pagination key | `meta` | `pagination` | Use `meta` |
| totalPages | camelCase | snake_case | Use camelCase |
| hasNextPage | camelCase | has_next | Use camelCase |
| hasPreviousPage | camelCase | has_prev | Use camelCase |

#### 2.3 Profile Endpoint

**Bun-Hono:** NOT IMPLEMENTED (profile embedded in user)

**Go Project:** Has separate /profile endpoint - Should consider removing or keeping

#### 2.4 Activity Logs

**Bun-Hono:** NOT IMPLEMENTED

**Go Project:** Has /activity-logs endpoint - Keep as is (additional feature)

---

### 3. PRODUCT SERVICE

#### 3.1 GET /products/:id Response (Single Product)

**Bun-Hono (With Variants):**
```json
{
  "success": true,
  "message": "Product fetched successfully",
  "data": {
    "id": "uuid",
    "name": "T-Shirt",
    "price": { "min": 100, "max": 200, "display": "$100 - $200" },
    "stock": 50,
    "has_variant": true,
    "owner_id": "user-uuid",
    "attributes": [
      {
        "id": "attr-1",
        "name": "Color",
        "values": ["Red", "Blue"],
        "display_order": 1
      }
    ],
    "variants": [
      {
        "id": "var-1",
        "sku": "TSHIRT-RED-S",
        "price": 100,
        "stock_quantity": 25,
        "available_stock": 20,
        "is_active": true,
        "attribute_values": { "Color": "Red" }
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "deleted_at": null
  },
  "meta": null
}
```

**Bun-Hono (Without Variants):**
```json
{
  "success": true,
  "message": "Product fetched successfully",
  "data": {
    "id": "uuid",
    "name": "Simple Product",
    "price": { "min": 100, "max": 100, "display": "$100" },
    "stock": 50,
    "has_variant": false,
    "owner_id": "user-uuid",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "deleted_at": null
  },
  "meta": null
}
```

**Go Project (Current):**
```json
{
  "id": "uuid",
  "name": "Simple Product",
  "price": 100.00,
  "stock": 50,
  "owner_id": "user-uuid",
  "has_variant": false,
  "images": "http://...",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Differences:**
| Field | Bun-Hono | Go | Action |
|-------|----------|-----|--------|
| Root structure | success/data/meta | Direct fields | Wrap in standardized response |
| price | PriceRange object | Number | Change to PriceRange |
| has_variant | boolean | boolean | Keep |
| images | Not present | Present | Remove (not in Bun) |
| attributes | Present if has_variant | Not present | Add |
| variants | Present if has_variant | Not present | Add |

#### 3.2 GET /products Response (List Products)

**Bun-Hono:**
```json
{
  "success": true,
  "message": "Products fetched successfully",
  "data": [...products],
  "meta": {
    "includeDeleted": false,
    "onlyDeleted": false,
    "search": null,
    "priceRange": { "min": null, "max": null },
    "page": 1,
    "limit": 10,
    "total": 100,
    "totalPages": 10,
    "hasNextPage": true,
    "hasPreviousPage": false
  }
}
```

**Go Project (Current):**
```json
{
  "products": [...],
  "total": 100,
  "page": 1,
  "limit": 10,
  "total_pages": 10
}
```

**Differences:**
| Field | Bun-Hono | Go | Action |
|-------|----------|-----|--------|
| Root structure | success/data/meta | products/total/page... | Use standardized response |
| List key | data | products | Use data |
| Pagination key | meta | (flat) | Use meta |
| totalPages | camelCase | snake_case | Use camelCase |
| hasNextPage | camelCase | Not present | Add |
| hasPreviousPage | camelCase | Not present | Add |

#### 3.3 POST /products (Create Product)

**Bun-Hono:**
- Supports attributes and variants in creation

**Go Project (Current):**
- Does NOT support attributes and variants

**Action:** Add variant/attribute support

---

## Test Impact Analysis ⚠️

**IMPORTANT:** Each response change will require updating corresponding test assertions.

### Auth Service Tests (15+ assertions to update)

| File | Changes Needed |
|------|----------------|
| `internal/auth/delivery/handler_test.go` | `access_token` → `token`, wrap in response |
| `internal/auth/usecase/auth_usecase_test.go` | Auth response structure changes |
| `internal/auth/repository/user_repository_test.go` | Session.Token updates |

**Key test lines:**
- Lines 76-77, 301: `access_token`, `refresh_token` assertions
- Lines 499, 518, 535, 1071, 1744, 1829, 1856, 1858, 2730: refresh token tests

### User Service Tests

| File | Changes Needed |
|------|----------------|
| `internal/user/delivery/handler_test.go` | Profile removal, camelCase, pagination |
| `internal/user/usecase/user_usecase_test.go` | Profile field changes |
| `internal/user/repository/user_repository_test.go` | Pagination (16 occurrences) |

### Product Service Tests (25+ assertions)

| File | Changes Needed |
|------|----------------|
| `internal/product/delivery/handler_test.go` | HasVariant, variants, attributes, price format |
| `internal/product/usecase/product_usecase_test.go` | HasVariant updates |
| `internal/product/repository/product_repository_test.go` | HasVariant tests |

### Unaffected Tests
- `pkg/validator/*_test.go` - No changes needed
- `pkg/middleware/*_test.go` - No changes needed
- `pkg/eventbus/*_test.go` - No changes needed

---

## Implementation Plan

### ⚠️ Important: Run Tests After Each Phase

After each phase, run:
```bash
go test ./...  # Verify tests pass
```

If tests fail, update the corresponding test assertions to match new response format.

### Phase 1: Standardize Response Wrapper (High Priority)

Create a unified response wrapper for all endpoints:

```go
// Standard API Response
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Meta    interface{} `json:"meta,omitempty"`
}
```

**Files to modify:**
- [ ] `pkg/utils/response.go` - Create standardized response helpers

**Tests to update after this phase:**
- All handler tests will need wrapper assertion updates

### Phase 2: Auth Service Updates (High Priority)

**2.1 Update Login/Register Response:**
- [ ] Simplify to single token (remove refresh_token from response)
- [ ] Remove extra user fields (created_at, updated_at, etc.)
- [ ] Wrap in standardized response

**Files to modify:**
- [ ] `internal/auth/dto/response.go`

**Tests to update:**
- [ ] `internal/auth/delivery/handler_test.go` - Lines 76-77, 301, 499, 518, 535, etc.
- [ ] `internal/auth/usecase/auth_usecase_test.go`
- [ ] `internal/auth/repository/user_repository_test.go`

### Phase 3: User Service Updates (Medium Priority)

**3.1 Update User Response:**
- [ ] Remove profile from user response (or make optional)
- [ ] Use camelCase for all fields
- [ ] Wrap in standardized response

**Files to modify:**
- [ ] `internal/user/dto/response.go`

**Tests to update:**
- [ ] `internal/user/delivery/handler_test.go`
- [ ] `internal/user/usecase/user_usecase_test.go`
- [ ] `internal/user/repository/user_repository_test.go`

### Phase 4: Product Service Updates (High Priority)

**4.1 Implement Variants & Attributes:**
- [ ] Add attributes to product response
- [ ] Add variants to product response
- [ ] Change price to PriceRange object

**4.2 Update Product Response:**
- [ ] Remove `images` field (not in Bun-Hono)
- [ ] Use camelCase
- [ ] Wrap in standardized response

**Files to modify:**
- [ ] `internal/product/dto/response.go`
- [ ] `internal/product/domain/product.go`
- [ ] `internal/product/domain/variant.go`
- [ ] `internal/product/domain/attribute.go`

**Tests to update:**
- [ ] `internal/product/delivery/handler_test.go` - HasVariant, variants, attributes, price
- [ ] `internal/product/usecase/product_usecase_test.go` - HasVariant assertions
- [ ] `internal/product/repository/product_repository_test.go` - HasVariant tests (25+ assertions)

### Phase 5: Test Updates (Medium Priority)

**IMPORTANT:** This phase is for fixing any remaining test failures.

- [ ] Run `go test ./...` to identify failing tests
- [ ] Update all test assertions to match new response format
- [ ] Add tests for new variant/attribute functionality

---

## Priority Order

1. **P0 - Critical:**
   - Product variants/attributes implementation
   - Standardized response wrapper

2. **P1 - High:**
   - Auth response alignment
   - Product response alignment

3. **P2 - Medium:**
   - User response alignment
   - Pagination alignment

4. **P3 - Low:**
   - Activity logs (Go has extra feature, keep as is)

---

## Notes

- Go project has some extra fields that Bun-Hono doesn't have (e.g., `images`, `profile`)
- Bun-Hono uses camelCase, Go uses snake_case
- Bun-Hono wraps all responses in `success/data/meta` structure
- Go should adopt the standardized wrapper for consistency

---

## Files to Modify Summary

| Service | File | Changes |
|---------|------|---------|
| Common | pkg/utils/response.go | Add standardized response helpers |
| Auth | internal/auth/dto/response.go | Simplify auth response |
| User | internal/user/dto/response.go | Remove profile, update pagination |
| Product | internal/product/dto/response.go | Add variants, change price format |
| Product | internal/product/domain/*.go | Add variant/attribute domain models |
