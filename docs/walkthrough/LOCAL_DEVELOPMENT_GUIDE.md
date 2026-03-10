# Local Development Guide (Step-by-Step)

This guide is designed for developers coming from Node.js who want to run and test this Go microservices project locally, service by service.

Last synchronized with codebase: **March 10, 2026**.

---

## Table of Contents

1. [What You Are Running](#1-what-you-are-running)
2. [Prerequisites](#2-prerequisites)
3. [Important Configuration Behavior](#3-important-configuration-behavior)
4. [Choose Redis Mode (Local or Docker)](#4-choose-redis-mode-local-or-docker)
5. [Prepare PostgreSQL](#5-prepare-postgresql)
6. [Install Project Dependencies](#6-install-project-dependencies)
7. [Database Workflow (SQL Migration First)](#7-database-workflow-sql-migration-first)
8. [Run Each Service Locally](#8-run-each-service-locally)
9. [API Testing Flow (Practical)](#9-api-testing-flow-practical)
10. [Verify Redis Streams Are Working](#10-verify-redis-streams-are-working)
11. [Monitoring (Prometheus/Grafana)](#11-monitoring-prometheusgrafana)
12. [Troubleshooting](#12-troubleshooting)
13. [Quick Command Cheat Sheet](#13-quick-command-cheat-sheet)

---

## 1. What You Are Running

This repo currently has 3 Go services:

- `cmd/service-auth` (default local port `3100`)
- `cmd/service-user` (default local port `3101`)
- `cmd/service-product` (default local port `3102`)

Current DB architecture in this branch:

- **One shared PostgreSQL database**: `microservices_db`
- SQL migrations in `migrations/*.sql` are the source of truth
- Seeder in `cmd/db-seed` inserts starter data

Core infra:

- PostgreSQL (local recommended)
- Redis (local OR Docker)
- Optional: Redis Insight + Prometheus + Grafana (Docker)

---

## 2. Prerequisites

Required:

- Go >= `1.23`
- PostgreSQL >= `15`
- Redis >= `7` (if running locally)
- Docker + Docker Compose v2 (if running Redis/monitoring via Docker)
- `make`
- `curl`

Check versions:

```bash
go version
psql --version
redis-server --version || true
docker --version
docker compose version
make --version
curl --version
```

Optional but useful:

- `jq` (for parsing JSON in terminal)
- `httpie`

Install `jq` (macOS):

```bash
brew install jq
```

---

## 3. Important Configuration Behavior

### 3.1 How config is loaded

Config loader behavior:

1. Load `configs/<APP_ENV>.yaml`
2. Then apply environment variable overrides

Examples:

- `APP_ENV=local` -> `configs/local.yaml`
- `APP_ENV=user-local` -> `configs/user-local.yaml`
- `APP_ENV=product-local` -> `configs/product-local.yaml`

### 3.2 Very important `.env` behavior in this repo

`pkg/utils/LoadEnv()` currently loads `.env` and sets env vars directly.
Because of this, if `.env` contains `APP_ENV=local`, it can override your per-command `APP_ENV` when running services.

For multi-service development, **recommended**:

1. Open `.env`
2. Comment out service-scoped keys:
   - `APP_NAME`
   - `APP_ENV`
   - `SERVER_PORT`
   - `STREAMS_CONSUMER_GROUP`
   - `STREAMS_CONSUMER_NAME`
3. Keep shared keys active (DB, Redis, JWT, etc)

Example:

```dotenv
# APP_NAME=service-auth
# APP_ENV=local
# SERVER_PORT=3100
# STREAMS_CONSUMER_GROUP=service-auth
# STREAMS_CONSUMER_NAME=auth-1

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=microservices_db
DB_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379
```

If you skip this, user/product services may accidentally boot with `local` config and collide on port `3100`.

---

## 4. Choose Redis Mode (Local or Docker)

You can use either approach.

### Option A: Use local Redis service

Start local Redis (macOS Homebrew):

```bash
brew services start redis
redis-cli ping
```

Expected:

```text
PONG
```

### Option B: Use Redis in Docker (recommended if you do not want local Redis install)

```bash
docker compose -f deployments/docker-compose.yml up -d redis redis-insight
docker exec go-microservices-redis redis-cli ping
```

Expected:

```text
PONG
```

Redis Insight UI:

- URL: `http://localhost:5540`
- Host: `host.docker.internal` (macOS) or `localhost`
- Port: `6379`

---

## 5. Prepare PostgreSQL

Use local PostgreSQL for this workflow.

### 5.1 Start PostgreSQL

macOS (Homebrew):

```bash
brew services list | grep postgresql
brew services start postgresql@15
```

### 5.2 Verify connection

```bash
pg_isready -h localhost -p 5432
```

Expected:

```text
localhost:5432 - accepting connections
```

### 5.3 Check DB credentials match your config

Review these files:

- `configs/local.yaml`
- `configs/user-local.yaml`
- `configs/product-local.yaml`

All should point to:

- host: `localhost`
- port: `5432`
- name: `microservices_db`
- user/password matching your local PostgreSQL

If your local user is not `postgres`, update YAML or set env vars.

---

## 6. Install Project Dependencies

From repo root:

```bash
make deps
make build
```

What this does:

- `make deps` -> `go mod download` + `go mod tidy`
- `make build` -> builds all services into `bin/`

---

## 7. Database Workflow (SQL Migration First)

This project is configured for SQL-migration-first development.

### 7.1 Create DB

```bash
make db-create
```

### 7.2 Run all migrations

```bash
make db-migrate
```

Expected style output:

- `Applied N migration(s) using action 'up-all'`
- list of applied files under `migrations/`

### 7.3 Seed data (optional but recommended for fast testing)

```bash
make db-seed
```

Seeder inserts/updates:

- admin user
- regular user
- products
- attributes/variants for selected products

### 7.4 Migration management commands

Apply one migration file:

```bash
make db-migrate-up-one
```

Rollback one migration file (latest only):

```bash
make db-migrate-down-one
```

Rollback all:

```bash
make db-migrate-down-all
```

Create new sequential migration template:

```bash
make db-migrate-create name=add_status_to_products
```

### 7.5 Safety behavior on destructive rollback

Down migrations are guarded:

- If a down action would drop table/column with data, command fails by default

Force override (if you accept data loss):

```bash
MIGRATION_FORCE=1 make db-migrate-down-one
```

Rollback order is always latest-first (stack behavior), not arbitrary middle rollback.

---

## 8. Run Each Service Locally

Open **3 terminal tabs/windows** from project root.

### Terminal 1: Auth service

```bash
APP_ENV=local make run-service-auth
```

Expected log includes routes like:

- `/health`
- `/auth/register`
- `/auth/login`
- `/admin/users`

### Terminal 2: User service

```bash
APP_ENV=user-local make run-service-user
```

Expected log includes routes like:

- `/health`
- `/started`
- `/api/v1/users`
- `/api/v1/activity-logs`

### Terminal 3: Product service

```bash
APP_ENV=product-local make run-service-product
```

Expected log includes routes like:

- `/health`
- `/products`
- `/products/:id`
- `/products/:id/stock`

### Verify all health checks

```bash
curl http://localhost:3100/health
curl http://localhost:3101/health
curl http://localhost:3102/health
```

Expected JSON style:

```json
{"status":"ok","service":"service-auth"}
```

Note: service name differs per service.

---

## 9. API Testing Flow (Practical)

Below is a practical sequence you can run manually.

## 9.1 If you seeded already, use default credentials

After `make db-seed`:

- admin: `admin@example.com` / `Admin123!`
- user: `user@example.com` / `User123!`

### 9.1.1 Login seeded user

```bash
curl -s -X POST http://localhost:3100/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "User123!"
  }'
```

If you have `jq`, capture token/user id:

```bash
LOGIN_JSON=$(curl -s -X POST http://localhost:3100/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"User123!"}')

TOKEN=$(echo "$LOGIN_JSON" | jq -r '.data.token')
USER_ID=$(echo "$LOGIN_JSON" | jq -r '.data.user.id')

echo "TOKEN length: ${#TOKEN}"
echo "USER_ID: $USER_ID"
```

### 9.1.2 Verify auth/me

```bash
curl -s http://localhost:3100/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

### 9.1.3 List users from user service

```bash
curl -s http://localhost:3101/api/v1/users
```

### 9.1.4 Create a product

```bash
curl -s -X POST http://localhost:3102/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"NodeJS to Go Journey Tee\",
    \"price\": 22.99,
    \"stock\": 20,
    \"ownerId\": \"$USER_ID\",
    \"images\": \"https://example.com/tee.jpg\"
  }"
```

### 9.1.5 List products

```bash
curl -s http://localhost:3102/products
```

### 9.1.6 Update stock

Use product ID from create/list response:

```bash
PRODUCT_ID="<replace-with-product-id>"

curl -s -X PUT http://localhost:3102/products/$PRODUCT_ID/stock \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"stock":15}'
```

## 9.2 If you do not seed, register first

Register:

```bash
curl -s -X POST http://localhost:3100/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "username": "johndoe",
    "password": "MyP@ssw0rd123",
    "name": "John Doe"
  }'
```

Then login and continue same flow.

---

## 10. Verify Redis Streams Are Working

After register/login/create/update actions, verify stream data.

Important:

- `service-auth` publishes auth events to `auth:events`
- `service-user` consumes `auth:events` and writes into `user_activity_logs`
- If `service-user` is not running, stream entries still exist, but no activity log rows are created
- On startup, `service-user` now logs `Starting auth events consumer ...`

### 10.1 List keys

If Redis in Docker:

```bash
docker exec go-microservices-redis redis-cli KEYS "*"
```

If local Redis:

```bash
redis-cli KEYS "*"
```

### 10.2 Inspect main streams

Known stream names in code (`pkg/eventbus/options.go`):

- `auth:events`
- `users:events`
- `products:events`

Check stream metadata:

```bash
redis-cli XINFO STREAM auth:events
redis-cli XINFO STREAM users:events
redis-cli XINFO STREAM products:events
```

Show latest entries:

```bash
redis-cli XREVRANGE auth:events + - COUNT 5
redis-cli XREVRANGE users:events + - COUNT 5
redis-cli XREVRANGE products:events + - COUNT 5
```

Equivalent for Docker Redis:

```bash
docker exec go-microservices-redis redis-cli XREVRANGE products:events + - COUNT 5
```

### 10.3 Quick interpretation

You should see message IDs like `174...-0` with fields such as:

- `id`
- `type`
- `source`
- `timestamp`
- `payload`

If stream exists but no new entries, trigger events again:

- login/register on auth service
- create/update product on product service

---

## 11. Monitoring (Prometheus/Grafana)

Optional monitoring stack:

```bash
docker compose -f deployments/docker-compose.yml --profile monitoring up -d
```

UI URLs:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (`admin` / `admin`)
- Alertmanager: `http://localhost:9093`

### 11.1 Important target note

Current `deployments/monitoring/prometheus.yml` uses container-host targets by default.
If Go services are running on host machine, update targets to:

- `host.docker.internal:3100`
- `host.docker.internal:3101`
- `host.docker.internal:3102`

Then restart Prometheus:

```bash
docker compose -f deployments/docker-compose.yml --profile monitoring restart prometheus
```

### 11.2 Quick metrics checks

```bash
curl http://localhost:3100/metrics
curl http://localhost:3101/metrics
curl http://localhost:3102/metrics
```

Look for metrics families such as:

- `http_requests_total`
- `http_request_duration_seconds`
- `redis_publish_total`

---

## 12. Troubleshooting

### 12.1 `service-user` / `service-product` runs on wrong port

Cause:

- `.env` still forcing `APP_ENV=local` and/or `SERVER_PORT=3100`

Fix:

- comment out service-scoped keys in `.env` (section 3)
- restart service terminal

### 12.2 `bind: address already in use`

Find and kill process:

```bash
lsof -i :3100
lsof -i :3101
lsof -i :3102
kill -9 <PID>
```

### 12.3 `migration up-all failed: context canceled handling ...`

Checks:

```bash
pg_isready -h localhost -p 5432
make db-migrate
```

If still failing:

- ensure DB credentials in config/env are correct
- ensure database exists (`make db-create`)

### 12.4 Seeder says table missing

Run setup in order:

```bash
make db-create
make db-migrate
make db-seed
```

### 12.5 Down migration blocked by safety check

Expected when data exists in dropped table/column.

Force only if you are okay with data loss:

```bash
MIGRATION_FORCE=1 make db-migrate-down-one
```

### 12.6 Redis stream seems empty

Checklist:

1. Redis ping is `PONG`
2. At least one event-producing API was executed
3. Read with correct stream names (`auth:events`, `users:events`, `products:events`)
4. Use `XINFO STREAM <name>` to confirm stream exists

### 12.7 I changed schema and not reflected in DB

This project is SQL-migration-first.
Do not rely on runtime schema auto migration.

Workflow:

1. add new sequential SQL migration in `migrations/`
2. run `make db-migrate`

---

## 13. Quick Command Cheat Sheet

### First-time setup

```bash
make deps
make build

# Redis (docker option)
docker compose -f deployments/docker-compose.yml up -d redis redis-insight

# DB
make db-create
make db-migrate
make db-seed
```

### Run services

```bash
APP_ENV=local make run-service-auth
APP_ENV=user-local make run-service-user
APP_ENV=product-local make run-service-product
```

### Health checks

```bash
curl http://localhost:3100/health
curl http://localhost:3101/health
curl http://localhost:3102/health
```

### Migration operations

```bash
make db-migrate
make db-migrate-up-one
make db-migrate-down-one
make db-migrate-down-all
make db-migrate-create name=your_migration_name
```

### Redis stream checks

```bash
redis-cli XINFO STREAM products:events
redis-cli XREVRANGE products:events + - COUNT 5
```

---

## Notes for Node.js Developers

If you are used to Drizzle/Sequelize style workflows:

- `migrations/*.sql` in this project is the equivalent migration log source
- `make db-migrate-create` is the helper to scaffold new migration file
- `make db-migrate` is equivalent to applying pending migrations
- `make db-migrate-down-one` is equivalent to rollback latest migration
- rollback safety is built in for destructive down migrations

If you want, next step we can add a small `scripts/smoke-test.sh` to automate:

1. login
2. create product
3. verify product list
4. verify Redis stream event
