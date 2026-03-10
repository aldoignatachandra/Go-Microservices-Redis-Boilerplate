# User Service

User service handles user management and activity logs, and can consume auth-domain events.

## Runtime Snapshot

- Service entrypoint: `cmd/service-user/main.go`
- Default local config file: `configs/user-local.yaml` (`APP_ENV=user-local`)
- Default local port: `3101`
- Database: `microservices_db`
- Swagger UI: `http://localhost:3101/swagger/index.html`

## Run Locally

From repository root:

```bash
APP_ENV=user-local go run ./cmd/service-user
```

or:

```bash
APP_ENV=user-local make run-service-user
```

## Required Core Environment Variables

These are commonly overridden in local/dev:

- `APP_ENV` (for config file selection)
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`
- `SERVER_PORT` (if overriding YAML port)

## HTTP Endpoints

### Health and Observability

- `GET /health`
- `GET /ready`
- `GET /live`
- `GET /started`
- `GET /admin/health`
- `GET /metrics` (when metrics enabled)
- `GET /swagger/index.html`

### User API

- `GET /api/v1/users`
- `GET /api/v1/users/:id`
- `POST /api/v1/users/:id/activate`
- `POST /api/v1/users/:id/deactivate`
- `DELETE /api/v1/users/:id`
- `POST /api/v1/users/:id/restore`
- `GET /api/v1/activity-logs`

## Notes

- Route definitions live in `internal/user/delivery/routes.go`.
- Health endpoints are registered in `cmd/service-user/main.go`.
- If you run multiple services locally, make sure `.env` is not forcing `APP_ENV=local` and `SERVER_PORT=3100`.
