# Auth Service

The authentication service handles user registration, login, JWT token management, and session tracking.

## Overview

This service is responsible for:
- User registration with password hashing (bcrypt)
- User authentication with email/password
- JWT access and refresh token generation
- Session management (stateful sessions in database)
- Publishing authentication events to Redis Streams

## Configuration

Set the following environment variables:

| Variable                  | Description                        | Default                               |
| :------------------------ | :--------------------------------- | :------------------------------------ |
| `APP_NAME`                | Application name                   | `auth-service`                        |
| `APP_ENV`                 | Environment (local, staging, prod) | `local`                               |
| `SERVER_PORT`             | HTTP server port                   | `8080`                                |
| `DB_HOST`                 | PostgreSQL host                    | `localhost`                           |
| `DB_PORT`                 | PostgreSQL port                    | `5432`                                |
| `DB_NAME`                 | Database name                      | `auth_db`                             |
| `DB_USER`                 | Database user                      | `postgres`                            |
| `DB_PASSWORD`             | Database password                  | `postgres`                            |
| `REDIS_HOST`              | Redis host                         | `localhost`                           |
| `REDIS_PORT`              | Redis port                         | `6379`                                |
| `AUTH_JWT_SECRET`         | JWT signing secret                 | **REQUIRED - Change in production!**  |
| `AUTH_JWT_EXPIRES_IN`     | Access token expiry                | `15m`                                 |
| `AUTH_JWT_REFRESH_EXPIRES_IN` | Refresh token expiry           | `168h` (7 days)                       |
| `AUTH_BCRYPT_COST`        | Bcrypt hashing cost                | `12`                                  |
| `LOG_LEVEL`               | Logging level                      | `debug`                               |
| `LOG_FORMAT`              | Logging format (console/json)      | `console`                             |

## Running the Service

### Local Development

```bash
# From project root
make run-auth

# Or directly
go run ./cmd/auth-service

# Or with specific port
SERVER_PORT=8080 go run ./cmd/auth-service
```

### With Docker

```bash
# Build and run
docker build -f deployments/docker/Dockerfile.auth -t auth-service:latest .
docker run -p 8080:8080 --env-file .env auth-service:latest

# Or with docker-compose
docker-compose -f deployments/docker-compose.yml up auth-service
```

### With Hot Reload

```bash
# Requires Air: go install github.com/air-verse/air@latest
air -c .air.toml
```

## API Endpoints

### Public Endpoints

#### Register User

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "role": "USER"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "USER",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Protected Endpoints

> All protected endpoints require the `Authorization: Bearer <token>` header.

#### Get Current User

```http
GET /auth/me
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "USER",
    "is_active": true
  }
}
```

#### Logout

```http
POST /auth/logout
Authorization: Bearer <access_token>
```

### Health Endpoints

```http
GET /health      # Basic health check
GET /ready       # Readiness probe (checks DB + Redis)
GET /live        # Liveness probe
GET /metrics     # Prometheus metrics
```

## Events Published

The auth service publishes the following events to Redis Streams:

| Event Type        | Stream         | Description                    |
| :---------------- | :------------- | :----------------------------- |
| `user.created`    | `auth:events`  | New user registered            |
| `user.logged_in`  | `auth:events`  | User logged in successfully    |
| `user.logged_out` | `auth:events`  | User logged out                |

### Event Payload Example

```json
{
  "id": "evt_uuid",
  "type": "user.created",
  "source": "auth-service",
  "timestamp": 1704067200000,
  "payload": {
    "user_id": "user_uuid",
    "email": "user@example.com",
    "role": "USER"
  }
}
```

## Architecture

```
cmd/auth-service/
├── main.go              # Entry point, DI setup
├── wire.go              # Wire dependency injection definitions
└── wire_gen.go          # Generated wire code

internal/auth/
├── domain/              # Core business entities
│   ├── user.go          # User entity with password hashing
│   ├── session.go       # Session entity for token tracking
│   ├── events.go        # Event type constants
│   └── errors.go        # Domain-specific errors
├── dto/                 # Data Transfer Objects
│   ├── request.go       # Input validation structs
│   └── response.go      # Output response structs
├── repository/          # Data access layer
│   ├── user_repository.go
│   └── session_repository.go
├── usecase/             # Business logic
│   └── auth_usecase.go  # Auth orchestration
└── delivery/            # HTTP layer
    ├── handler.go       # HTTP handlers
    ├── routes.go        # Route registration
    └── middleware.go    # Auth middleware
```

## Testing

```bash
# Run auth service tests
go test ./internal/auth/... -v

# Run with coverage
go test ./internal/auth/... -cover

# Run specific test
go test ./internal/auth/usecase/... -run TestLogin -v
```

## Error Codes

| Code                    | HTTP Status | Description                    |
| :---------------------- | :---------- | :----------------------------- |
| `VALIDATION_ERROR`      | 400         | Invalid request data           |
| `UNAUTHORIZED`          | 401         | Invalid or expired token       |
| `USER_NOT_FOUND`        | 404         | User does not exist            |
| `USER_ALREADY_EXISTS`   | 409         | Email already registered       |
| `INVALID_CREDENTIALS`   | 401         | Wrong email or password        |
| `TOKEN_EXPIRED`         | 401         | JWT token has expired          |
| `TOKEN_INVALID`         | 401         | JWT token is malformed         |
| `INTERNAL_ERROR`        | 500         | Server error                   |

## Security Considerations

1. **Password Hashing**: Uses bcrypt with cost 12 (configurable)
2. **JWT Signing**: HS256 algorithm with configurable secret
3. **Token Expiry**: Short-lived access tokens (15 min default), long-lived refresh tokens (7 days default)
4. **Session Tracking**: Stateful sessions stored in database for revocation capability
5. **Rate Limiting**: Redis-backed rate limiting on authentication endpoints

## Dependencies

- `github.com/gin-gonic/gin` - HTTP framework
- `gorm.io/gorm` - ORM
- `github.com/redis/go-redis/v9` - Redis client
- `go.uber.org/zap` - Structured logging
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `golang.org/x/crypto/bcrypt` - Password hashing
- `github.com/google/wire` - Dependency injection
