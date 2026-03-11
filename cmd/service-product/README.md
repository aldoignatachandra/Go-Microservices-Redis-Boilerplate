# Product Service

Product service is responsible for product CRUD and stock operations.

It owns:
- product create/read/update/delete/restore
- variant-aware stock update endpoint
- owner-aware access control for product resources
- product event publishing to Redis Streams

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

- Entrypoint: `cmd/service-product/main.go`
- Wire setup: `cmd/service-product/wire.go`
- Default local config file: `configs/product-local.yaml` (`APP_ENV=product-local`)
- Default local port: `3102`
- Default DB name: `microservices_db`
- Swagger UI: `http://localhost:3102/swagger/index.html`
- Published stream: `products:events`

---

## Architecture & Data Ownership

Core package structure for this service:
- Delivery: `internal/product/delivery`
- Use case: `internal/product/usecase`
- Repository: `internal/product/repository`
- Domain model: `internal/product/domain`

Primary tables used:
- `products`
- `product_variants`
- `product_attributes`

Behavioral highlights from current code:
- Product endpoints are JWT-protected.
- Read path (`GetProduct`, `ListProducts`) applies owner scoping for non-admins.
- Mutation paths (`Update/Delete/Restore/UpdateStock`) are owner-only.
- Admin can read other users' products, but cannot mutate them.
- Create/update with variants syncs parent product stock from total variant stock.
- Stock update endpoint supports variant-level reduction for variant products (`id` in body = variant id).

---

## Access Control Model

All `/api/v1/products/*` routes require valid Bearer JWT.

### Read access
- Admin: can read all products.
- Non-admin: can read only own products.

### Mutation access
- Owner-only for:
  - `PUT /api/v1/products/:id`
  - `DELETE /api/v1/products/:id`
  - `POST /api/v1/products/:id/restore`
  - `PUT /api/v1/products/:id/stock`

---

## Run Locally

From repository root:

```bash
APP_ENV=product-local make run-service-product
```

or:

```bash
APP_ENV=product-local go run ./cmd/service-product
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
| `APP_ENV` | Selects YAML config file | `product-local` |
| `SERVER_PORT` | HTTP port override | `3102` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `microservices_db` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_SECRET` | JWT verification secret | custom |

### Important `.env` note

`pkg/utils/LoadEnv()` force-loads `.env` at startup. If `.env` contains service-scoped values (especially `APP_ENV`), they can override command-line env values.

---

## HTTP API

### Primary endpoints

| Method | Endpoint | Description | Auth |
|---|---|---|---|
| GET | `/api/v1/products` | List products (owner-scoped for non-admin) | Bearer |
| GET | `/api/v1/products/:id` | Get product by id (owner-scoped for non-admin) | Bearer |
| POST | `/api/v1/products` | Create product | Bearer |
| PUT | `/api/v1/products/:id` | Update product (owner only) | Bearer |
| DELETE | `/api/v1/products/:id` | Delete product (owner only) | Bearer |
| POST | `/api/v1/products/:id/restore` | Restore product (owner only) | Bearer |
| PUT | `/api/v1/products/:id/stock` | Reduce stock (variant-aware, owner only) | Bearer |

Stock endpoint behavior:
- For simple products: body `{ "stock": <quantity_to_reduce> }`.
- For variant products: body `{ "id": "<variant_id>", "stock": <quantity_to_reduce> }`.
- Variant id must belong to the parent product id in path.

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

### 1. Login first (Auth service)

```bash
curl -X POST http://localhost:3100/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"User123!"}'
```

### 2. Create product

```bash
curl -X POST http://localhost:3102/api/v1/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -d '{
    "name":"Premium T-Shirt",
    "price":29.99,
    "stock":100,
    "ownerId":"USER_ID",
    "images":"https://example.com/tshirt.jpg"
  }'
```

### 3. Reduce stock quantity

```bash
curl -X PUT http://localhost:3102/api/v1/products/PRODUCT_ID/stock \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -d '{"stock":3}'
```

### 4. Reduce variant product stock

```bash
curl -X PUT http://localhost:3102/api/v1/products/PRODUCT_ID/stock \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -d '{"id":"VARIANT_ID","stock":3}'
```

### 5. List own products

```bash
curl http://localhost:3102/api/v1/products?page=1&limit=10 \
  -H "Authorization: Bearer ACCESS_TOKEN"
```

---

## Event Streams

### Published streams
- `products:events`

Published event types:
- `product.created`
- `product.updated`
- `product.deleted`
- `product.restored`
- `product.stock_updated`

Metadata such as correlation/request identifiers is attached when available.

### Consumed streams
- None in current implementation.

---

## Rate Limiting

When Redis rate limiting is enabled, route-level limits are configured in `cmd/service-product/main.go`:
- `/api/v1/products` -> 120 req / 60s
- `/api/v1/products/:id` -> 10 req / 60s
- `/api/v1/products/:id/restore` -> 10 req / 60s
- `/api/v1/products/:id/stock` -> 30 req / 60s

A fallback global limit from config is also applied by middleware.

---

## Health & Observability

- Request ID + structured logging middleware enabled by default.
- Metrics middleware enabled when `metrics.enabled=true`.
- Swagger docs source: `cmd/service-product/docs`.

Useful checks:

```bash
curl http://localhost:3102/health
curl http://localhost:3102/ready
curl http://localhost:3102/metrics
```

---

## Troubleshooting

### 403 on update/delete/restore even with valid token
- Usually owner mismatch (resource belongs to different `owner_id`).
- Current policy intentionally enforces owner-only mutation.

### Admin cannot update another user's product
- Expected by design in current usecase logic.

### List endpoint returns fewer products than expected
- For non-admin users, `owner_id` is forced to caller id in usecase.

### Stock update fails on variant product
- Ensure path id is the parent product id.
- Provide variant id in request body field `id`.
- Ensure variant id belongs to that parent product.

### No product events seen in Redis
- Verify Redis connectivity and check `products:events` stream:

```bash
docker exec -it go-microservices-redis redis-cli XRANGE products:events - +
```
