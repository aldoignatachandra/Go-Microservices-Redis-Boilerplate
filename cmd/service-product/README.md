# Product Service

Product service manages products, product variants, attributes, stock operations, and product events.

## Runtime Snapshot

- Service entrypoint: `cmd/service-product/main.go`
- Default local config file: `configs/product-local.yaml` (`APP_ENV=product-local`)
- Default local port: `3102`
- Database: `microservices_db`
- Swagger UI: `http://localhost:3102/swagger/index.html`

## Run Locally

From repository root:

```bash
APP_ENV=product-local go run ./cmd/service-product
```

or:

```bash
APP_ENV=product-local make run-service-product
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

### Product API

- `GET /products`
- `GET /products/:id`
- `POST /products`
- `PUT /products/:id`
- `DELETE /products/:id`
- `POST /products/:id/restore`
- `PUT /products/:id/stock`

## Notes

- Product route definitions live in `internal/product/delivery/routes.go`.
- Health endpoints are registered in `cmd/service-product/main.go`.
- Product API base path in this Go service is `/products` (not `/api/v1/products`).
