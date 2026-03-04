# Product Service

The product service handles product catalog management, including product variants and inventory tracking.

## Overview

This service is responsible for:
- Product CRUD operations (Create, Read, Update, Delete)
- Product variant management (size, color, etc.)
- Inventory quantity tracking
- Soft delete and restore functionality
- Publishing product events to Redis Streams

## Configuration

Set the following environment variables:

| Variable                  | Description                        | Default                               |
| :------------------------ | :--------------------------------- | :------------------------------------ |
| `APP_NAME`                | Application name                   | `product-service`                     |
| `APP_ENV`                 | Environment (local, staging, prod) | `local`                               |
| `SERVER_PORT`             | HTTP server port                   | `8082`                                |
| `DB_HOST`                 | PostgreSQL host                    | `localhost`                           |
| `DB_PORT`                 | PostgreSQL port                    | `5432`                                |
| `DB_NAME`                 | Database name                      | `product_db`                          |
| `DB_USER`                 | Database user                      | `postgres`                            |
| `DB_PASSWORD`             | Database password                  | `postgres`                            |
| `REDIS_HOST`              | Redis host                         | `localhost`                           |
| `REDIS_PORT`              | Redis port                         | `6379`                                |
| `LOG_LEVEL`               | Logging level                      | `debug`                               |
| `LOG_FORMAT`              | Logging format (console/json)      | `console`                             |

## Running the Service

### Local Development

```bash
# From project root
make run-product

# Or directly
go run ./cmd/product-service

# Or with specific port
SERVER_PORT=8082 go run ./cmd/product-service
```

### With Docker

```bash
# Build and run
docker build -f deployments/docker/Dockerfile.product -t product-service:latest .
docker run -p 8082:8082 --env-file .env product-service:latest

# Or with docker-compose
docker-compose -f deployments/docker-compose.yml --profile full up product-service
```

### With Hot Reload

```bash
# Requires Air: go install github.com/air-verse/air@latest
air -c .air.toml
```

## API Endpoints

### Public Endpoints

#### List Products

```http
GET /api/v1/products
```

**Query Parameters:**
| Parameter     | Type    | Description                    |
| :------------ | :------ | :----------------------------- |
| `page`        | int     | Page number (default: 1)       |
| `limit`       | int     | Items per page (default: 20)   |
| `search`      | string  | Search by name or SKU           |
| `category`    | string  | Filter by category              |
| `min_price`   | float   | Minimum price filter            |
| `max_price`   | float   | Maximum price filter            |
| `is_active`   | bool    | Filter active products          |

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "uuid",
        "name": "Product Name",
        "sku": "SKU-001",
        "description": "Product description",
        "price": 99.99,
        "currency": "USD",
        "quantity": 100,
        "category": "electronics",
        "is_active": true,
        "variants": [
          {
            "id": "uuid",
            "name": "Size",
            "value": "Large",
            "price_adjustment": 10.00
          }
        ],
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20,
    "total_pages": 5
  }
}
```

#### Get Product by ID

```http
GET /api/v1/products/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Product Name",
    "sku": "SKU-001",
    "description": "Product description",
    "price": 99.99,
    "currency": "USD",
    "quantity": 100,
    "category": "electronics",
    "is_active": true,
    "images": [
      "https://example.com/image1.jpg"
    ],
    "variants": [],
    "metadata": {
      "weight": "1.5kg",
      "dimensions": "10x20x30cm"
    }
  }
}
```

#### Get Product by SKU

```http
GET /api/v1/products/sku/:sku
```

### Protected Endpoints

> All protected endpoints require the `Authorization: Bearer <token>` header.

#### Create Product

```http
POST /api/v1/products
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "New Product",
  "sku": "SKU-002",
  "description": "Product description",
  "price": 149.99,
  "currency": "USD",
  "quantity": 50,
  "category": "electronics",
  "is_active": true,
  "images": ["https://example.com/image1.jpg"],
  "variants": [
    {
      "name": "Color",
      "value": "Red",
      "price_adjustment": 0
    }
  ]
}
```

**Response:** `201 Created`

#### Update Product

```http
PUT /api/v1/products/:id
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Updated Product Name",
  "price": 129.99,
  "quantity": 75
}
```

#### Delete Product (Soft Delete)

```http
DELETE /api/v1/products/:id
Authorization: Bearer <access_token>
```

**Query Parameters:**
| Parameter | Type    | Description                              |
| :-------- | :------ | :--------------------------------------- |
| `force`   | bool    | Hard delete if `true` (default: false)   |

#### Restore Product

```http
POST /api/v1/products/:id/restore
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Restored Product",
    "restored_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Update Inventory

```http
PATCH /api/v1/products/:id/inventory
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "quantity": 150,
  "operation": "set"
}
```

**Operations:**
- `set` - Set absolute quantity
- `add` - Add to current quantity
- `subtract` - Subtract from current quantity

### Health Endpoints

```http
GET /health      # Basic health check
GET /ready       # Readiness probe (checks DB + Redis)
GET /live        # Liveness probe
GET /metrics     # Prometheus metrics
```

## Events Published

The product service publishes the following events to Redis Streams:

| Event Type          | Stream            | Description                    |
| :------------------ | :---------------- | :----------------------------- |
| `product.created`   | `products:events` | New product created            |
| `product.updated`   | `products:events` | Product information updated    |
| `product.deleted`   | `products:events` | Product soft deleted           |
| `product.restored`  | `products:events` | Product restored               |
| `inventory.updated` | `products:events` | Inventory quantity changed     |

### Event Payload Example

```json
{
  "id": "evt_uuid",
  "type": "product.created",
  "source": "product-service",
  "timestamp": 1704067200000,
  "payload": {
    "product_id": "product_uuid",
    "name": "New Product",
    "sku": "SKU-002",
    "price": 149.99,
    "quantity": 50
  }
}
```

## Architecture

```
cmd/product-service/
├── main.go              # Entry point, DI setup
├── wire.go              # Wire dependency injection definitions
└── wire_gen.go          # Generated wire code

internal/product/
├── domain/              # Core business entities
│   ├── product.go       # Product entity with variants
│   ├── variant.go       # Product variant entity
│   ├── events.go        # Event type constants
│   └── errors.go        # Domain-specific errors
├── dto/                 # Data Transfer Objects
│   ├── request.go       # Input validation structs
│   └── response.go      # Output response structs
├── repository/          # Data access layer
│   └── product_repository.go
├── usecase/             # Business logic
│   └── product_usecase.go
└── delivery/            # HTTP layer
    ├── handler.go       # HTTP handlers
    └── routes.go        # Route registration
```

## Product Entity

```go
type Product struct {
    Model                          // ID, CreatedAt, UpdatedAt, DeletedAt
    Name        string             `gorm:"type:varchar(255);not null"`
    SKU         string             `gorm:"type:varchar(100);uniqueIndex;not null"`
    Description string             `gorm:"type:text"`
    Price       float64            `gorm:"type:decimal(10,2);not null"`
    Currency    string             `gorm:"type:varchar(3);default:'USD'"`
    Quantity    int                `gorm:"type:integer;default:0"`
    Category    string             `gorm:"type:varchar(100)"`
    IsActive    bool               `gorm:"default:true"`
    Images      pq.StringArray     `gorm:"type:text[]"`
    Metadata    map[string]interface{} `gorm:"type:jsonb"`
    Variants    []ProductVariant   `gorm:"foreignKey:ProductID"`
    OwnerID     string             `gorm:"type:uuid;index"`
}
```

## Soft Delete Behavior

| Operation            | Description                              |
| :------------------- | :-------------------------------------- |
| **Soft Delete**      | Sets `deleted_at` timestamp (default)   |
| **Hard Delete**      | Permanently removes (use `force=true`)   |
| **Restore**          | Clears `deleted_at` timestamp            |

## Testing

```bash
# Run product service tests
go test ./internal/product/... -v

# Run with coverage
go test ./internal/product/... -cover

# Run specific test
go test ./internal/product/usecase/... -run TestCreateProduct -v
```

## Error Codes

| Code                    | HTTP Status | Description                    |
| :---------------------- | :---------- | :----------------------------- |
| `VALIDATION_ERROR`      | 400         | Invalid request data           |
| `UNAUTHORIZED`          | 401         | Invalid or expired token       |
| `FORBIDDEN`             | 403         | Insufficient permissions       |
| `PRODUCT_NOT_FOUND`     | 404         | Product does not exist         |
| `SKU_ALREADY_EXISTS`    | 409         | SKU already in use             |
| `INSUFFICIENT_INVENTORY`| 400         | Not enough inventory           |
| `PRODUCT_ALREADY_DELETED`| 400        | Product already soft deleted   |
| `PRODUCT_NOT_DELETED`   | 400         | Cannot restore non-deleted product |
| `INTERNAL_ERROR`        | 500         | Server error                   |

## Dependencies

- `github.com/gin-gonic/gin` - HTTP framework
- `gorm.io/gorm` - ORM with soft delete support
- `github.com/redis/go-redis/v9` - Redis client
- `go.uber.org/zap` - Structured logging
- `github.com/google/wire` - Dependency injection
