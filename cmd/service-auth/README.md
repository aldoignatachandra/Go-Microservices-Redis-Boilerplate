# Auth Service

Auth service handles registration, login, token lifecycle, user session management, and admin user operations.

## Runtime Snapshot

- Service entrypoint: `cmd/service-auth/main.go`
- Default local config file: `configs/local.yaml` (`APP_ENV=local`)
- Default local port: `3100`
- Database: `microservices_db`
- Swagger UI: `http://localhost:3100/swagger/index.html`

## Run Locally

From repository root:

```bash
APP_ENV=local go run ./cmd/service-auth
```

or:

```bash
APP_ENV=local make run-service-auth
```

## Required Core Environment Variables

These are commonly overridden in local/dev:

- `APP_ENV` (for config file selection)
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`
- `JWT_SECRET`, `JWT_EXPIRES_IN`, `JWT_REFRESH_EXPIRES_IN`
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

### Public Auth

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`

### Authenticated Auth

- `POST /auth/logout`
- `GET /auth/me`
- `POST /auth/change-password`

### Admin User Management

- `GET /admin/users`
- `GET /admin/users/:id`
- `DELETE /admin/users/:id`
- `POST /admin/users/:id/restore`

## Notes

- Route definitions live in `internal/auth/delivery/routes.go`.
- Health endpoints are registered in `cmd/service-auth/main.go`.
- If Swagger shows endpoints that return 404, regenerate docs with `make swagger` and verify route wiring in `main.go`.
