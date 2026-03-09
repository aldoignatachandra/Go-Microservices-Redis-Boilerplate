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
| `search`      | string  | Search by name                 |
| `owner_id`    | string  | Filter by owner                 |
| `status`      | string  | Filter by status (ACTIVE, INACTIVE) |

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "name": "Product Name",
      "price": {
        "min": 29.99,
        "max": 29.99,
        "display": "$29.99"
      },
      "stock": 100,
      "hasVariant": false,
      "ownerId": "user-uuid",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 500,
  "page": 1,
  "limit": 20,
  "totalPages": 25,
  "hasNextPage": true,
  "hasPreviousPage": false
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
    "price": {
      "min": 29.99,
      "max": 29.99,
      "display": "$29.99"
    },
    "stock": 100,
    "hasVariant": false,
    "ownerId": "user-uuid",
    "attributes": [],
    "variants": [],
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

### Protected Endpoints

> All protected endpoints require the `Authorization: Bearer <token>` header.

#### Create Product

```http
POST /api/v1/products
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Product Name",
  "price": 29.99,
  "stock": 100,
  "ownerId": "user-uuid",
  "images": "https://example.com/image.jpg"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Product Name",
    "price": {
      "min": 29.99,
      "max": 29.99,
      "display": "$29.99"
    },
    "stock": 100,
    "hasVariant": false,
    "ownerId": "user-uuid",
    "images": "https://example.com/image.jpg",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

#### Create Product with Variants

```http
POST /api/v1/products
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "T-Shirt",
  "price": 29.99,
  "stock": 100,
  "ownerId": "user-uuid",
  "attributes": [
    {
      "name": "Color",
      "values": ["Red", "Blue", "Green"],
      "displayOrder": 1
    },
    {
      "name": "Size",
      "values": ["S", "M", "L", "XL"],
      "displayOrder": 2
    }
  ],
  "variants": [
    {
      "name": "T-Shirt - Red - S",
      "sku": "TSHIRT-RED-S",
      "price": 29.99,
      "stockQuantity": 25,
      "isActive": true,
      "attributeValues": {
        "Color": "Red",
        "Size": "S"
      }
    }
  ]
}
```

#### Update Product

```http
PUT /api/v1/products/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Product Name",
  "price": 39.99,
  "stock": 150
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Updated Product Name",
    "price": {
      "min": 39.99,
      "max": 39.99,
      "display": "$39.99"
    },
    "stock": 150,
    "hasVariant": false,
    "ownerId": "user-uuid",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-02T00:00:00Z"
  }
}
```

#### Update Stock

```http
PUT /api/v1/products/:id/stock
Authorization: Bearer <token>
Content-Type: application/json

{
  "stock": 200
}
```

**Response:**
```json
{
  "success": true,
  "message": "Stock updated successfully"
}
```

#### Delete Product (Soft Delete)

```http
DELETE /api/v1/products/:id
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "message": "Product deleted successfully"
}
```

#### Restore Product

```http
POST /api/v1/products/:id/restore
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "message": "Product restored successfully"
}
```

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
