# Auth Service

Auth service is responsible for authentication and account lifecycle management in this project.

It owns:
- login, logout, refresh, and password change flows
- server-side session tracking (`user_sessions`) that backs JWT revocation behavior
- admin-only user management endpoints (list/get/delete/restore/register)
- publishing auth/user lifecycle events to Redis Streams
- consuming all project streams for observability logging

---

## Table of Contents

- [Runtime Snapshot](#runtime-snapshot)
- [Architecture & Data Ownership](#architecture--data-ownership)
- [Access Control Model](#access-control-model)
- [Run Locally](#run-locally)
- [Configuration](#configuration)
- [HTTP API](#http-api)
- [Example Request Flows](#example-request-flows)
- [Event Streams](#event-streams)
- [Rate Limiting](#rate-limiting)
- [Health & Observability](#health--observability)
- [Troubleshooting](#troubleshooting)

---

## Runtime Snapshot

- Entrypoint: `cmd/service-auth/main.go`
- Wire setup: `cmd/service-auth/wire.go`
- Default local config file: `configs/local.yaml` (`APP_ENV=local`)
- Default local port: `3100`
- Default DB name: `microservices_db`
- Swagger UI: `http://localhost:3100/swagger/index.html`
- Primary stream publish target: `auth:events`
- Active consumers at startup:
  - `auth:events`
  - `users:events`
  - `products:events`

---

## Architecture & Data Ownership

Core package structure for this service:
- Delivery: `internal/auth/delivery`
- Use case: `internal/auth/usecase`
- Repositories: `internal/auth/repository`
- Domain model: `internal/auth/domain`

Primary tables used:
- `users`
- `user_sessions`

Behavioral highlights from current code:
- **Single-session policy on login**: existing sessions are deleted before new session creation.
- **Refresh rotates session state**: refresh revokes prior sessions and creates a new session+token pair.
- **Password change revokes sessions**: changing password forces re-authentication.
- **Logout validates active session**: repeated logout with stale JWT is rejected.

---

## Access Control Model

### Public routes
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`

### Authenticated routes
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/change-password`

### Admin-only routes
- `POST /api/v1/auth/register`
- `GET /api/v1/users`
- `GET /api/v1/users/:id`
- `DELETE /api/v1/users/:id`
- `POST /api/v1/users/:id/restore`

Important:
- Register endpoint is admin-only and creates `USER` role accounts.
- For local first-run, use seeded admin credentials or pre-existing admin account.

---

## Run Locally

From repository root:

```bash
APP_ENV=local make run-service-auth
```

or:

```bash
APP_ENV=local go run ./cmd/service-auth
```

Recommended DB/bootstrap sequence before first run:

```bash
make db-create
make db-migrate
make db-seed
```

Default seeded admin credentials:
- Email: `admin@example.com`
- Password: `Admin123!`

---

## Configuration

### Common environment overrides

| Variable | Purpose | Typical local value |
|---|---|---|
| `APP_ENV` | Selects YAML config file | `local` |
| `SERVER_PORT` | HTTP port override | `3100` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `microservices_db` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_SECRET` | JWT signing secret | custom |
| `JWT_EXPIRES_IN` | Access token lifetime | `24h` |
| `JWT_REFRESH_EXPIRES_IN` | Refresh token lifetime | `168h` |
| `STREAMS_CONSUMER_GROUP` | Consumer group override | `service-auth` |
| `STREAMS_CONSUMER_NAME` | Consumer name override | `auth-1` |

### Important `.env` note

`pkg/utils/LoadEnv()` force-loads `.env` at startup. If `.env` contains service-scoped values (especially `APP_ENV`), they can override command-line env values.

---

## HTTP API

### Primary endpoints

| Method | Endpoint | Description | Auth |
|---|---|---|---|
| POST | `/api/v1/auth/login` | Authenticate user | Public |
| POST | `/api/v1/auth/refresh` | Rotate token pair | Public |
| POST | `/api/v1/auth/logout` | Revoke user sessions | Bearer |
| GET | `/api/v1/auth/me` | Current user profile | Bearer |
| POST | `/api/v1/auth/change-password` | Change password + revoke sessions | Bearer |

### Admin endpoints

| Method | Endpoint | Description | Auth |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Create USER account | Bearer + Admin |
| GET | `/api/v1/users` | List users | Bearer + Admin |
| GET | `/api/v1/users/:id` | Get user by ID | Bearer + Admin |
| DELETE | `/api/v1/users/:id` | Soft/hard delete user | Bearer + Admin |
| POST | `/api/v1/users/:id/restore` | Restore soft-deleted user | Bearer + Admin |

### Health endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | Public health |
| GET | `/ready` | Readiness probe |
| GET | `/live` | Liveness probe |
| GET | `/started` | Startup probe |
| GET | `/admin/health` | Detailed dependency health |
| GET | `/metrics` | Prometheus metrics (if enabled) |

---

## Example Request Flows

### 1. Login and get profile

```bash
# Login
curl -X POST http://localhost:3100/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"Admin123!"}'

# Use returned access token
curl http://localhost:3100/api/v1/auth/me \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

### 2. Register user (admin-only)

```bash
curl -X POST http://localhost:3100/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN" \
  -d '{
    "email":"newuser@example.com",
    "username":"newuser",
    "name":"New User",
    "password":"Password123!",
    "confirmPassword":"Password123!"
  }'
```

### 3. Change password

```bash
curl -X POST http://localhost:3100/api/v1/auth/change-password \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -d '{
    "old_password":"OldPassword123!",
    "new_password":"NewPassword123!"
  }'
```

---

## Event Streams

### Published streams
- Stream: `auth:events`

Representative published event types:
- `user.created`
- `user.updated`
- `user.deleted`
- `user.restored`
- `user.logged_in`
- `user.logged_out`
- `user.refreshed_token`

### Consumed streams
- `auth:events`
- `users:events`
- `products:events`

Current consumer behavior in this service is observability-focused (logs consumed event metadata).

---

## Rate Limiting

When Redis rate limiting is enabled, route-level limits are configured in `cmd/service-auth/main.go`:
- `/api/v1/auth/login` -> 10 req / 60s
- `/api/v1/auth/logout` -> 30 req / 60s
- `/api/v1/auth/register` -> 10 req / 60s

A fallback global limit from config is also applied by middleware.

---

## Health & Observability

- Request ID + structured logging middleware enabled by default.
- Metrics middleware enabled when `metrics.enabled=true`.
- Swagger docs source: `cmd/service-auth/docs`.

Useful checks:

```bash
curl http://localhost:3100/health
curl http://localhost:3100/ready
curl http://localhost:3100/metrics
```

---

## Troubleshooting

### Login returns unauthorized but token seems valid
- Cause: server-side session check failed (session revoked/expired).
- Check table: `user_sessions` and verify session id from JWT claim.

### Register returns forbidden
- Cause: route is admin-only.
- Use seeded admin login first, then call register with admin token.

### Refresh fails after logout/password change
- Expected behavior. Refresh token is invalidated when sessions are revoked.

### No events visible
- Verify Redis connectivity and stream names.
- Check consumer group/consumer name in config and logs at startup.
