# User Service

User service is responsible for user administration and activity log querying/ingestion.

It owns:
- user read/list/admin lifecycle operations
- activity log retrieval endpoint
- auth-event consumption and activity log persistence
- publishing user lifecycle events to `users:events`

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

- Entrypoint: `cmd/service-user/main.go`
- Wire setup: `cmd/service-user/wire.go`
- Default local config file: `configs/user-local.yaml` (`APP_ENV=user-local`)
- Default local port: `3101`
- Default DB name: `microservices_db`
- Swagger UI: `http://localhost:3101/swagger/index.html`
- Consumes stream: `auth:events`
- Publishes stream: `users:events`

---

## Architecture & Data Ownership

Core package structure for this service:
- Delivery: `internal/user/delivery`
- Use case: `internal/user/usecase`
- Repositories: `internal/user/repository`
- Domain model: `internal/user/domain`

Primary tables used:
- `users`
- `user_activity_logs`

Behavioral highlights from current code:
- `/api/v1/users/:id` can be accessed by self or admin.
- list/activity/admin lifecycle routes are admin-only.
- self-modification guard exists for admin lifecycle actions:
  - admin cannot activate/deactivate/delete/restore own account (`forbidSelfAdminAction`).

---

## Access Control Model

All `/api/v1/*` routes require valid Bearer JWT.

### Self or admin
- `GET /api/v1/users/:id`

### Admin-only
- `GET /api/v1/users`
- `GET /api/v1/activity-logs`
- `POST /api/v1/users/:id/activate`
- `POST /api/v1/users/:id/deactivate`
- `DELETE /api/v1/users/:id`
- `POST /api/v1/users/:id/restore`

---

## Run Locally

From repository root:

```bash
APP_ENV=user-local make run-service-user
```

or:

```bash
APP_ENV=user-local go run ./cmd/service-user
```

Recommended bootstrap before first run:

```bash
make db-create
make db-migrate
make db-seed
```

---

## Configuration

### Common environment overrides

| Variable | Purpose | Typical local value |
|---|---|---|
| `APP_ENV` | Selects YAML config file | `user-local` |
| `SERVER_PORT` | HTTP port override | `3101` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `microservices_db` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_SECRET` | JWT verification secret | custom |
| `STREAMS_CONSUMER_GROUP` | Consumer group override | `service-user` |
| `STREAMS_CONSUMER_NAME` | Consumer name override | `user-1` |

### Important `.env` note

`pkg/utils/LoadEnv()` force-loads `.env` at startup. If `.env` contains service-scoped values (especially `APP_ENV`), they can override command-line env values.

---

## HTTP API

### Primary endpoints

| Method | Endpoint | Description | Auth |
|---|---|---|---|
| GET | `/api/v1/users/:id` | Get user profile by id (self/admin) | Bearer |
| GET | `/api/v1/users` | List users (pagination/filter) | Bearer + Admin |
| GET | `/api/v1/activity-logs` | List activity logs | Bearer + Admin |
| POST | `/api/v1/users/:id/activate` | Activate user | Bearer + Admin |
| POST | `/api/v1/users/:id/deactivate` | Deactivate user | Bearer + Admin |
| DELETE | `/api/v1/users/:id` | Delete user (soft by default) | Bearer + Admin |
| POST | `/api/v1/users/:id/restore` | Restore user | Bearer + Admin |

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

### 1. Login (Auth service), then get own profile from user service

```bash
# login on auth service first
curl -X POST http://localhost:3100/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"User123!"}'

# fetch own user profile from user service
curl http://localhost:3101/api/v1/users/USER_ID \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

### 2. Admin list users

```bash
curl http://localhost:3101/api/v1/users?page=1&limit=20 \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

### 3. Admin fetch activity logs

```bash
curl "http://localhost:3101/api/v1/activity-logs?page=1&limit=20&action=user.logged_in" \
  -H "Authorization: Bearer ADMIN_ACCESS_TOKEN"
```

---

## Event Streams

### Consumed streams
- `auth:events`

Current behavior:
- auth events are translated into activity logs via `LogActivity` use case.
- event metadata and request context are persisted in `user_activity_logs` details.

### Published streams
- `users:events`

Published event types from user lifecycle commands:
- `user.deleted`
- `user.restored`

---

## Rate Limiting

When Redis rate limiting is enabled, route-level limits are configured in `cmd/service-user/main.go`:
- `/api/v1/users` -> 120 req / 60s
- `/api/v1/users/:id` -> 5 req / 60s

A fallback global limit from config is also applied by middleware.

---

## Health & Observability

- Request ID + structured logging middleware enabled by default.
- Metrics middleware enabled when `metrics.enabled=true`.
- Swagger docs source: `cmd/service-user/docs`.

Useful checks:

```bash
curl http://localhost:3101/health
curl http://localhost:3101/ready
curl http://localhost:3101/metrics
```

---

## Troubleshooting

### 403 when admin tries to deactivate/delete self
- Expected: self-admin lifecycle changes are explicitly blocked in handler guard.

### 403 when non-admin reads other user
- Expected: only own profile (or admin) can access `/api/v1/users/:id`.

### No activity logs created from auth events
- Verify Redis connectivity.
- Confirm auth service publishes to `auth:events`.
- Check user service startup logs for consumer group creation and consumption errors.
