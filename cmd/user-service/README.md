# User Service

The user service handles user profile management, activity logging, and user administration.

## Overview

This service is responsible for:
- User profile CRUD operations
- Activity log tracking and retrieval
- User activation/deactivation
- Soft delete and restore functionality
- Consuming user events from Auth service

## Configuration

Set the following environment variables:

| Variable                  | Description                        | Default                               |
| :------------------------ | :--------------------------------- | :------------------------------------ |
| `APP_NAME`                | Application name                   | `user-service`                        |
| `APP_ENV`                 | Environment (local, staging, prod) | `local`                               |
| `SERVER_PORT`             | HTTP server port                   | `8081`                                |
| `DB_HOST`                 | PostgreSQL host                    | `localhost`                           |
| `DB_PORT`                 | PostgreSQL port                    | `5432`                                |
| `DB_NAME`                 | Database name                      | `user_db`                             |
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
make run-user

# Or directly
go run ./cmd/user-service

# Or with specific port
SERVER_PORT=8081 go run ./cmd/user-service
```

### With Docker

```bash
# Build and run
docker build -f deployments/docker/Dockerfile.user -t user-service:latest .
docker run -p 8081:8081 --env-file .env user-service:latest

# Or with docker-compose
docker-compose -f deployments/docker-compose.yml --profile full up user-service
```

### With Hot Reload

```bash
# Requires Air: go install github.com/air-verse/air@latest
air -c .air.toml
```

## API Endpoints

### All endpoints require authentication

> All endpoints require the `Authorization: Bearer <token>` header.

### Profile Endpoints

#### Get User Profile

```http
GET /api/v1/users/:id/profile
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": "uuid",
    "first_name": "John",
    "last_name": "Doe",
    "avatar": "https://example.com/avatar.jpg",
    "bio": "Software Developer",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Update User Profile

```http
PUT /api/v1/users/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "John",
  "last_name": "Doe",
  "avatar": "https://example.com/avatar.jpg",
  "bio": "Updated bio"
}
```

### User Management Endpoints

#### List Users

```http
GET /api/v1/users?page=1&limit=20&role=USER&search=john
Authorization: Bearer <access_token>
```

**Query Parameters:**
| Parameter        | Type    | Description                    |
| :--------------- | :------ | :----------------------------- |
| `page`           | int     | Page number (default: 1)       |
| `limit`          | int     | Items per page (default: 20)   |
| `role`           | string  | Filter by role (USER, ADMIN)   |
| `search`         | string  | Search by email or name        |
| `include_deleted`| bool    | Include soft-deleted users     |
| `only_deleted`   | bool    | Only show deleted users        |

**Response:**
```json
{
  "success": true,
  "data": {
    "users": [...],
    "total": 100,
    "page": 1,
    "limit": 20,
    "total_pages": 5
  }
}
```

#### Get User by ID

```http
GET /api/v1/users/:id
Authorization: Bearer <access_token>
```

#### Activate User

```http
POST /api/v1/users/:id/activate
Authorization: Bearer <access_token>
```

#### Deactivate User

```http
POST /api/v1/users/:id/deactivate
Authorization: Bearer <access_token>
```

#### Delete User (Soft Delete)

```http
DELETE /api/v1/users/:id
Authorization: Bearer <access_token>
```

**Query Parameters:**
| Parameter | Type | Description                    |
| :-------- | :--- | :----------------------------- |
| `force`   | bool | Hard delete if true            |

#### Restore Deleted User

```http
POST /api/v1/users/:id/restore
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "is_active": true
    },
    "message": "User restored successfully"
  }
}
```

### Activity Log Endpoints

#### Get Activity Logs

```http
GET /api/v1/activity-logs?user_id=uuid&action=login&page=1&limit=20
Authorization: Bearer <access_token>
```

**Query Parameters:**
| Parameter | Type    | Description                    |
| :-------- | :------ | :----------------------------- |
| `user_id` | string  | Filter by user ID              |
| `action`  | string  | Filter by action type          |
| `resource`| string  | Filter by resource type        |
| `page`    | int     | Page number                    |
| `limit`   | int     | Items per page                 |

### Health Endpoints

```http
GET /health      # Basic health check
GET /ready       # Readiness probe (checks DB + Redis)
GET /live        # Liveness probe
GET /metrics     # Prometheus metrics
```

## Events Consumed

The user service consumes the following events from Redis Streams:

| Event Type        | Stream         | Action                          |
| :---------------- | :------------- | :------------------------------ |
| `user.created`    | `auth:events`  | Create user profile             |
| `user.logged_in`  | `auth:events`  | Log login activity              |
| `user.logged_out` | `auth:events`  | Log logout activity             |

## Events Published

| Event Type              | Stream          | Description                    |
| :---------------------- | :-------------- | :----------------------------- |
| `profile.updated`       | `user:events`   | User profile was updated       |
| `user.activated`        | `user:events`   | User account was activated     |
| `user.deactivated`      | `user:events`   | User account was deactivated   |
| `user.deleted`          | `user:events`   | User was soft deleted          |
| `user.restored`         | `user:events`   | User was restored              |
| `activity.created`      | `activity:log`  | Activity log entry created     |

## Architecture

```
cmd/user-service/
├── main.go              # Entry point, DI setup
├── wire.go              # Wire dependency injection definitions
└── wire_gen.go          # Generated wire code

internal/user/
├── domain/              # Core business entities
│   ├── user.go          # User entity with soft delete
│   ├── profile.go       # Profile entity
│   ├── activity_log.go  # Activity log entity
│   ├── events.go        # Event type constants
│   └── errors.go        # Domain-specific errors
├── dto/                 # Data Transfer Objects
│   ├── request.go       # Input validation structs
│   └── response.go      # Output response structs
├── repository/          # Data access layer
│   ├── user_repository.go
│   └── activity_repository.go
├── usecase/             # Business logic
│   └── user_usecase.go  # User orchestration
└── delivery/            # HTTP layer
    ├── handler.go       # HTTP handlers
    ├── routes.go        # Route registration
    └── middleware.go    # Auth middleware
```

## Soft Delete (Paranoid Mode)

This service implements soft delete functionality:

- **Soft Delete**: Sets `deleted_at` timestamp, record remains in database
- **Hard Delete**: Permanently removes record (use `force=true`)
- **Restore**: Clears `deleted_at` timestamp

### Query Behavior

| Query Option         | Behavior                        |
| :------------------- | :------------------------------ |
| Default              | Excludes deleted records        |
| `include_deleted=true` | Includes all records          |
| `only_deleted=true`  | Only returns deleted records    |

See [GORM Best Practices](../../docs/standardization/GORM_BEST_PRACTICES.md) for more details.

## Testing

```bash
# Run user service tests
go test ./internal/user/... -v

# Run with coverage
go test ./internal/user/... -cover

# Run specific test
go test ./internal/user/usecase/... -run TestUpdateProfile -v
```

## Activity Log Types

| Action                | Description                              |
| :-------------------- | :-------------------------------------- |
| `profile_updated`     | User updated their profile               |
| `user_activated`      | User account was activated               |
| `user_deactivated`    | User account was deactivated             |
| `user_deleted`        | User account was soft deleted            |
| `user_restored`       | User account was restored                |

## Error Codes

| Code                    | HTTP Status | Description                    |
| :---------------------- | :---------- | :----------------------------- |
| `VALIDATION_ERROR`      | 400         | Invalid request data           |
| `UNAUTHORIZED`          | 401         | Invalid or expired token       |
| `FORBIDDEN`             | 403         | Insufficient permissions       |
| `USER_NOT_FOUND`        | 404         | User does not exist            |
| `PROFILE_NOT_FOUND`     | 404         | Profile does not exist         |
| `USER_ALREADY_DELETED`  | 400         | User already soft deleted      |
| `USER_NOT_DELETED`      | 400         | Cannot restore non-deleted user|
| `INTERNAL_ERROR`        | 500         | Server error                   |

## Dependencies

- `github.com/gin-gonic/gin` - HTTP framework
- `gorm.io/gorm` - ORM with soft delete support
- `github.com/redis/go-redis/v9` - Redis client
- `go.uber.org/zap` - Structured logging
- `github.com/google/wire` - Dependency injection
