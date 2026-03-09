# 🚀 Local Development Walkthrough

> **A comprehensive, step-by-step guide to running and testing the Go Microservices Redis Pub/Sub Boilerplate locally.**
>
> This guide is written for backend developers who are new to Go. Every command is explained.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Prerequisites](#2-prerequisites)
3. [Project Structure Quick Reference](#3-project-structure-quick-reference)
4. [Step 1: Clone and Install Dependencies](#step-1-clone-and-install-dependencies)
5. [Step 2: Start Infrastructure (Docker)](#step-2-start-infrastructure-docker)
6. [Step 3: Create PostgreSQL Databases Locally](#step-3-create-postgresql-databases-locally)
7. [Step 4: Configure Environment for Each Service](#step-4-configure-environment-for-each-service)
8. [Step 5: Run the Auth Service](#step-5-run-the-auth-service)
9. [Step 6: Run the User Service](#step-6-run-the-user-service)
10. [Step 7: Run the Product Service](#step-7-run-the-product-service)
11. [Step 8: Test API Endpoints](#step-8-test-api-endpoints)
12. [Step 9: Monitor with Prometheus & Grafana](#step-9-monitor-with-prometheus--grafana)
13. [Step 10: View Redis Streams Events](#step-10-view-redis-streams-events)
14. [Troubleshooting](#troubleshooting)
15. [Useful Makefile Commands](#useful-makefile-commands)

---

## 1. Architecture Overview

This boilerplate consists of **3 independent microservices** communicating via **Redis Streams** (event-driven):

```
┌────────────────────┐     ┌────────────────────┐     ┌────────────────────┐
│   AUTH SERVICE     │     │   USER SERVICE     │     │  PRODUCT SERVICE   │
│   Port: 3100       │     │   Port: 3101       │     │   Port: 3102       │
│   DB: auth_db      │     │   DB: user_db      │     │   DB: product_db   │
└────────┬───────────┘     └────────┬───────────┘     └────────┬───────────┘
         │                          │                          │
         └──────────────────────────┼──────────────────────────┘
                                    │
                          ┌─────────▼──────────┐
                          │   REDIS STREAMS    │
                          │   (Event Bus)      │
                          │   Port: 6379       │
                          └────────────────────┘
```

**What each service does:**

| Service     | Port   | Database     | Responsibility                                        |
| :---------- | :----- | :----------- | :---------------------------------------------------- |
| **Auth**    | `3100` | `auth_db`    | User registration, login, JWT tokens, password change |
| **User**    | `3101` | `user_db`    | User profiles, user management, activity logs         |
| **Product** | `3102` | `product_db` | Product CRUD, stock management, product events        |

**Infrastructure services (Docker):**

| Service           | Port   | Purpose                         |
| :---------------- | :----- | :------------------------------ |
| **Redis**         | `6379` | Event bus (Redis Streams)       |
| **Redis Insight** | `5540` | Redis GUI for debugging streams |
| **Prometheus**    | `9090` | Metrics collection              |
| **Grafana**       | `3000` | Metrics visualization           |
| **Alertmanager**  | `9093` | Alert management                |

---

## 2. Prerequisites

Before starting, make sure you have the following installed on your machine:

### Required

| Tool                   | Version | Check Command            | Install                                                       |
| :--------------------- | :------ | :----------------------- | :------------------------------------------------------------ |
| **Go**                 | ≥ 1.23  | `go version`             | [golang.org/dl](https://golang.org/dl/)                       |
| **Docker**             | ≥ 24    | `docker --version`       | [docker.com](https://www.docker.com/products/docker-desktop/) |
| **Docker Compose**     | ≥ 2.0   | `docker compose version` | Included with Docker Desktop                                  |
| **PostgreSQL**         | ≥ 15    | `psql --version`         | `brew install postgresql@15` (macOS)                          |
| **Make**               | any     | `make --version`         | Pre-installed on macOS                                        |
| **curl** or **httpie** | any     | `curl --version`         | Pre-installed on macOS                                        |

### Optional (but recommended)

| Tool              | Purpose            | Install                                                                 |
| :---------------- | :----------------- | :---------------------------------------------------------------------- |
| **Air**           | Hot reload for Go  | `go install github.com/air-verse/air@latest`                            |
| **golangci-lint** | Go linter          | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| **Wire**          | DI code generation | `go install github.com/google/wire/cmd/wire@latest`                     |

> **💡 Node.js Analogy:**
>
> - `go version` is like `node --version`
> - `go install` is like `npm install -g`
> - `go run ./cmd/auth-service` is like `npx ts-node src/index.ts`

---

## 3. Project Structure Quick Reference

```
go-microservices-redis-pubsub-boilerplate/
├── cmd/                     ← Entry points for each service (like src/index.ts)
│   ├── auth-service/        ← main.go, wire.go, wire_gen.go
│   ├── user-service/        ← main.go, wire.go, wire_gen.go
│   └── product-service/     ← main.go, wire.go, wire_gen.go
├── internal/                ← Private business logic (Clean Architecture)
│   ├── auth/                ← domain/, dto/, repository/, usecase/, delivery/
│   ├── user/                ← domain/, dto/, repository/, usecase/, delivery/
│   ├── product/             ← domain/, dto/, repository/, usecase/, delivery/
│   └── common/              ← Shared constants, errors, middleware
├── pkg/                     ← Shared libraries ("Platform Kit")
│   ├── config/              ← Viper-based config loader
│   ├── database/            ← PostgreSQL + Redis connection helpers
│   ├── eventbus/            ← Redis Streams producer/consumer
│   ├── logger/              ← Zap structured logging
│   ├── metrics/             ← Prometheus metrics
│   ├── middleware/          ← HTTP middleware (auth, rate-limit, etc.)
│   ├── server/              ← Graceful server + health checks
│   └── utils/               ← JWT, hashing, HTTP response helpers
├── configs/                 ← YAML config files per service
│   ├── local.yaml           ← Auth service config (APP_ENV=local)
│   ├── user-local.yaml      ← User service config
│   └── product-local.yaml   ← Product service config
├── deployments/             ← Docker & K8s infrastructure
│   ├── docker-compose.yml   ← All containers
│   ├── docker/              ← Dockerfiles per service
│   └── monitoring/          ← Prometheus, Grafana, Alertmanager configs
├── scripts/
│   └── init-db.sql          ← Creates auth_db, user_db, product_db
├── Makefile                 ← 315 lines of build/test/deploy commands
└── .env                     ← Environment variable overrides
```

---

## Step 1: Clone and Install Dependencies

```bash
# Navigate to the project directory
cd ~/Desktop/Self\ Project/Project-Golang/go-microservices-redis-pubsub-boilerplate

# Download all Go module dependencies
go mod download

# Tidy up (remove unused, add missing)
go mod tidy
```

Or, using the Makefile shortcut:

```bash
make deps
```

> **💡 What this does:** `go mod download` is like `npm install`. It reads `go.mod` (like `package.json`) and downloads all dependencies to `$GOPATH/pkg/mod/` (like `node_modules/`, but global).

### Verify the build compiles

```bash
make build
```

This builds all 3 services into the `bin/` directory. If it compiles without errors, you're good to go.

---

## Step 2: Start Infrastructure (Docker)

We'll run **Redis** and **monitoring stack** in Docker, but keep **PostgreSQL** and the **Go services** running locally so you can see the logs directly in your terminal.

### 2a. Start Redis + Redis Insight only

```bash
# From the project root directory, start only Redis and Redis Insight
docker compose -f deployments/docker-compose.yml up -d redis redis-insight
```

**Verify Redis is running:**

```bash
# Ping Redis (should respond with PONG)
docker exec go-microservices-redis redis-cli ping
```

Expected output:

```
PONG
```

**Access Redis Insight GUI:**

Open your browser and go to: **http://localhost:5540**

In Redis Insight, add a new database connection:

- **Host:** `host.docker.internal` (or `localhost` if on Linux)
- **Port:** `6379`
- **Name:** `Go Microservices Redis`

### 2b. Start Monitoring Stack (Prometheus + Grafana + Alertmanager)

```bash
# Start the monitoring profile
docker compose -f deployments/docker-compose.yml --profile monitoring up -d
```

> **Note:** The `--profile monitoring` flag tells Docker Compose to also start the Prometheus, Grafana, and Alertmanager containers which are defined under the `monitoring` profile.

**Verify monitoring is running:**

| Service      | URL                   | Expected                 |
| :----------- | :-------------------- | :----------------------- |
| Prometheus   | http://localhost:9090 | Prometheus web UI        |
| Grafana      | http://localhost:3000 | Login page (admin/admin) |
| Alertmanager | http://localhost:9093 | Alertmanager web UI      |

---

## Step 3: Create PostgreSQL Databases Locally

Since we're running PostgreSQL **locally** (not in Docker), we need to create the 3 databases manually.

### 3a. Make sure PostgreSQL is running

```bash
# Check if PostgreSQL is running (macOS with Homebrew)
brew services list | grep postgresql

# If not running, start it:
brew services start postgresql@15
```

### 3b. Create the databases

```bash
# Connect to your local PostgreSQL instance
psql -U postgres

# If you get a "role does not exist" error, try:
# psql -U $(whoami) -d postgres
```

Once in the `psql` shell, run:

```sql
-- Create the 3 databases (one per service)
CREATE DATABASE auth_db;
CREATE DATABASE user_db;
CREATE DATABASE product_db;

-- Enable UUID extension in each database
\c auth_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c user_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c product_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Verify all databases exist
\l

-- Exit psql
\q
```

> **💡 Why 3 databases?** Each microservice owns its own database — this is the **Database per Service** pattern. It ensures services are truly independent and can't accidentally read/write each other's data.

### 3c. Alternatively, use the init script

If your local PostgreSQL user is `postgres`, you can run the project's init script:

```bash
psql -U postgres -f scripts/init-db.sql
```

---

## Step 4: Configure Environment for Each Service

The config system uses **Viper** (a Go config library similar to `dotenv` + `config` in Node.js). It loads config from:

1. **YAML file** in `configs/` directory (selected by `APP_ENV` environment variable)
2. **Environment variables** (override YAML values)

### How config file selection works

The config loader (`pkg/config/loader.go`) does this:

```
APP_ENV=local → reads configs/local.yaml
APP_ENV=user-local → reads configs/user-local.yaml
APP_ENV=product-local → reads configs/product-local.yaml
```

### Verify your YAML configs

The project already has 3 config files pre-configured for local development:

| Config File                  | Service | Port   | Database     |
| :--------------------------- | :------ | :----- | :----------- |
| `configs/local.yaml`         | Auth    | `3100` | `auth_db`    |
| `configs/user-local.yaml`    | User    | `3101` | `user_db`    |
| `configs/product-local.yaml` | Product | `3102` | `product_db` |

All configs point to `localhost:5432` (PostgreSQL) and `localhost:6379` (Redis).

### Verify PostgreSQL credentials

Open `configs/local.yaml` and make sure the `database` section matches your local PostgreSQL setup:

```yaml
database:
  host: "localhost"
  port: 5432
  name: "auth_db"
  user: "postgres" # ← Change if your pg user is different
  password: "postgres" # ← Change if your pg password is different
  sslmode: "disable"
```

> **⚠️ Common Issue:** On macOS, if you installed PostgreSQL via Homebrew, your default superuser is your macOS username (not `postgres`). You may need to update the `user` field in all 3 YAML configs, or create a `postgres` user:
>
> ```bash
> createuser -s postgres
> ```

---

## Step 5: Run the Auth Service

Open a **new terminal window/tab** (you'll need one per service for logs).

### 5a. Start the auth service

```bash
# From the project root
APP_ENV=local go run ./cmd/auth-service
```

Or using the Makefile:

```bash
make run-auth-service
```

### 5b. What you should see

```
{"level":"info","msg":"Starting auth service","host":"0.0.0.0","port":3100}
```

> **💡 What happened under the hood:**
>
> 1. Go compiled and ran `cmd/auth-service/main.go`
> 2. `config.Load("")` read `configs/local.yaml` (because `APP_ENV=local`)
> 3. Wire injected all dependencies (logger → database → redis → eventbus → usecase → handler)
> 4. **GORM auto-migrated** the auth database tables (users, sessions)
> 5. Gin HTTP server started listening on port `3100`

### 5c. Verify it's running

```bash
# Health check
curl http://localhost:3100/health

# Expected response:
# {"status":"ok","service":"auth-service"}
```

### 5d. Auth service endpoints

| Method | Endpoint                   | Auth Required | Description                 |
| :----- | :------------------------- | :------------ | :-------------------------- |
| POST   | `/auth/register`           | No            | Register new user           |
| POST   | `/auth/login`             | No            | Login and get JWT token     |
| POST   | `/auth/refresh`           | No            | Refresh JWT token          |
| POST   | `/auth/logout`            | ✅ JWT        | Logout (invalidate session)|
| GET    | `/auth/me`                | ✅ JWT        | Get current user info      |
| POST   | `/auth/change-password`  | ✅ JWT        | Change password           |
| GET    | `/admin/users`            | ✅ Admin      | List all users            |
| GET    | `/admin/users/:id`        | ✅ Admin      | Get user by ID            |
| DELETE | `/admin/users/:id`        | ✅ Admin      | Delete user               |
| POST   | `/admin/users/:id/restore`| ✅ Admin      | Restore deleted user      |
| GET    | `/health`                 | No            | Health check              |
| GET    | `/ready`                  | No            | Readiness probe           |
| GET    | `/live`                   | No            | Liveness probe            |
| GET    | `/metrics`                | No            | Prometheus metrics        |

---

## Step 6: Run the User Service

Open a **second terminal window/tab**.

### 6a. Start the user service

```bash
# From the project root
APP_ENV=user-local go run ./cmd/user-service
```

Or:

```bash
make run-user-service
# Note: you'll need to set APP_ENV=user-local beforehand or it will default to local.yaml
```

> **⚠️ Important:** You MUST set `APP_ENV=user-local` so it reads `configs/user-local.yaml` (port 3101, database `user_db`). If you forget, it will try to use `configs/local.yaml` and conflict with the auth service on port 3100.

### 6b. Verify it's running

```bash
curl http://localhost:3101/health

# Expected: {"status":"ok","service":"user-service"}
```

### 6c. User service endpoints

| Method | Endpoint                       | Description          |
| :----- | :----------------------------- | :------------------- |
| GET    | `/api/v1/users`                | List all users       |
| GET    | `/api/v1/users/:id`            | Get user by ID       |
| POST   | `/api/v1/users/:id/activate`   | Activate user        |
| POST   | `/api/v1/users/:id/deactivate` | Deactivate user      |
| DELETE | `/api/v1/users/:id`            | Delete user          |
| POST   | `/api/v1/users/:id/restore`    | Restore deleted user |
| GET    | `/api/v1/activity-logs`        | Get activity logs    |
| GET    | `/health`                      | Health check         |
| GET    | `/metrics`                     | Prometheus metrics   |

---

## Step 7: Run the Product Service

Open a **third terminal window/tab**.

### 7a. Start the product service

```bash
# From the project root
APP_ENV=product-local go run ./cmd/product-service
```

### 7b. Verify it's running

```bash
curl http://localhost:3102/health

# Expected: {"status":"ok","service":"product-service"}
```

### 7c. Product service endpoints

| Method | Endpoint                | Description             |
| :----- | :---------------------- | :---------------------- |
| GET    | `/products`             | List products           |
| GET    | `/products/:id`         | Get product by ID       |
| POST   | `/products`             | Create product          |
| PUT    | `/products/:id`         | Update product          |
| DELETE | `/products/:id`         | Delete product          |
| POST   | `/products/:id/restore` | Restore deleted product |
| PUT    | `/products/:id/stock`   | Update product stock    |
| GET    | `/health`               | Health check            |
| GET    | `/metrics`              | Prometheus metrics      |

---

## Step 8: Test API Endpoints

Now that all 3 services are running, let's test the full API workflow.

### 8a. Register a new user (Auth Service)

```bash
curl -X POST http://localhost:3100/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "username": "johndoe",
    "password": "MyP@ssw0rd123",
    "name": "John Doe"
  }'
```

**Expected Response (201):**

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "uuid-here",
      "email": "john@example.com",
      "username": "johndoe",
      "name": "John Doe",
      "role": "USER",
      "createdAt": "2026-03-09T12:00:00Z",
      "updatedAt": "2026-03-09T12:00:00Z"
    }
  }
}
```

### 8b. Login (Auth Service)

```bash
curl -X POST http://localhost:3100/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "MyP@ssw0rd123"
  }'
```

**Expected Response (200):**

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "uuid-here",
      "email": "john@example.com",
      "username": "johndoe",
      "name": "John Doe",
      "role": "USER",
      "createdAt": "2026-03-09T12:00:00Z",
      "updatedAt": "2026-03-09T12:00:00Z"
    }
  }
}
```

> **📋 Save the `token`** — you'll need it for authenticated requests.

### 8c. Get Current User (Auth Service — Authenticated)

```bash
# Replace <ACCESS_TOKEN> with the token from the login response
curl http://localhost:3100/auth/me \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

### 8d. Create a Product (Product Service)

```bash
curl -X POST http://localhost:3102/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "name": "MacBook Pro M4",
    "price": 2499.99,
    "stock": 100,
    "ownerId": "<USER_ID>"
  }'
```

> **Note:** You'll need to get the `ownerId` from the `/auth/me` endpoint response.

### 8e. List Products (Product Service)

```bash
curl http://localhost:3102/products
```

### 8f. Update Stock (Product Service)

```bash
# Replace <PRODUCT_ID> with the ID from the create response
curl -X PUT http://localhost:3102/products/<PRODUCT_ID>/stock \
  -H "Content-Type: application/json" \
  -d '{
    "stock": 5
  }'
```

### 8g. List Users (User Service)

```bash
curl http://localhost:3101/api/v1/users
```

### 8h. Check Prometheus Metrics

```bash
# Auth service metrics
curl http://localhost:3100/metrics

# User service metrics
curl http://localhost:3101/metrics

# Product service metrics
curl http://localhost:3102/metrics
```

The output will be in Prometheus text format. Look for:

- `http_requests_total` — total HTTP requests by method, path, and status
- `http_request_duration_seconds` — request latency histogram
- `redis_publish_total` — events published to Redis Streams

---

## Step 9: Monitor with Prometheus & Grafana

### 9a. Update Prometheus targets for local services

The default `deployments/monitoring/prometheus.yml` assumes services run inside Docker with container names. Since we're running services locally, we need to update the scrape targets.

Create or edit a local prometheus override:

```bash
# Check current targets in Prometheus UI
open http://localhost:9090/targets
```

> **⚠️ Note:** The default prometheus.yml uses Docker container names (e.g., `auth-service:8080`). Since our services run locally on the host, Prometheus (running in Docker) needs to reach them via `host.docker.internal` on macOS.

To fix this temporarily, you can use the Prometheus **Targets** page (http://localhost:9090/targets) to verify which targets are UP/DOWN.

For a quick fix, you can update `deployments/monitoring/prometheus.yml`:

```yaml
# Replace the service targets with host.docker.internal
- job_name: "auth-service"
  static_configs:
    - targets: ["host.docker.internal:3100"]

- job_name: "user-service"
  static_configs:
    - targets: ["host.docker.internal:3101"]

- job_name: "product-service"
  static_configs:
    - targets: ["host.docker.internal:3102"]
```

Then restart Prometheus:

```bash
docker compose -f deployments/docker-compose.yml --profile monitoring restart prometheus
```

### 9b. Access Grafana

1. Open http://localhost:3000
2. Login with **admin** / **admin** (skip password change for dev)
3. Prometheus should already be configured as a data source
4. You can create dashboards or explore metrics via **Explore** → select `prometheus` → type a query like `http_requests_total`

### 9c. Useful Prometheus Queries

| Query                                                                      | What it shows                   |
| :------------------------------------------------------------------------- | :------------------------------ |
| `http_requests_total`                                                      | Total requests per service      |
| `rate(http_requests_total[5m])`                                            | Requests per second (5m window) |
| `http_request_duration_seconds_bucket`                                     | Latency distribution            |
| `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` | P95 latency                     |
| `redis_publish_total`                                                      | Events published to Redis       |

---

## Step 10: View Redis Streams Events

When you create a product or register a user, events are published to Redis Streams. You can observe them in two ways:

### 10a. Using Redis CLI

```bash
# Check which streams exist
docker exec go-microservices-redis redis-cli KEYS "*"

# Read all events from the product stream
docker exec go-microservices-redis redis-cli XRANGE products:events - +

# Read all events from the auth stream
docker exec go-microservices-redis redis-cli XRANGE auth:events - +

# Read all events from the user stream
docker exec go-microservices-redis redis-cli XRANGE users:events - +

# Watch events in real-time (like `tail -f`)
docker exec go-microservices-redis redis-cli MONITOR
```

### 10b. Using Redis Insight GUI

1. Open http://localhost:5540
2. Connect to your Redis instance
3. Navigate to the **Streams** section
4. You'll see streams like `products:events`, `auth:events`
5. Click on a stream to view its messages and consumer groups

### 10c. Understanding Event Flow

When you create a product, the following happens:

```
1. POST /products → Product Service Handler
2. Handler → Product UseCase (business logic)
3. UseCase → Product Repository (save to PostgreSQL)
4. UseCase → EventBus Producer (publish event to Redis Stream)
5. Redis Stream "products:events" receives:
   {
     "id": "unique-event-id",
     "type": "product.created",
     "source": "product-service",
     "timestamp": 1741152000,
     "payload": { "id": "...", "name": "MacBook Pro M4", ... }
   }
6. Any service with a Consumer for "products:events" will receive this event
```

---

## Troubleshooting

### ❌ "connection refused" when starting a service

**Cause:** PostgreSQL or Redis is not running.

```bash
# Check PostgreSQL
pg_isready -h localhost -p 5432

# Check Redis
docker exec go-microservices-redis redis-cli ping
```

### ❌ "role postgres does not exist"

**Cause:** On macOS with Homebrew PostgreSQL, the default superuser is your macOS username.

**Fix:** Create the postgres user:

```bash
createuser -s postgres
```

Or update the `database.user` field in all config files to your macOS username.

### ❌ "database auth_db does not exist"

**Cause:** You haven't created the databases yet.

**Fix:** See [Step 3: Create PostgreSQL Databases](#step-3-create-postgresql-databases-locally).

### ❌ "address already in use" on a port

**Cause:** Another service or process is using that port.

```bash
# Find what's using port 3100
lsof -i :3100

# Kill the process
kill -9 <PID>
```

### ❌ Services can't connect to Redis

**Cause:** Redis container might not be running.

```bash
# Check container status
docker ps | grep redis

# Restart if needed
docker compose -f deployments/docker-compose.yml up -d redis
```

### ❌ "yaml: line X: mapping values are not allowed in this context"

**Cause:** The wrong YAML config file is being loaded.

**Fix:** Make sure you're setting `APP_ENV` correctly:

```bash
# Auth service
APP_ENV=local go run ./cmd/auth-service

# User service (NOT just "local"!)
APP_ENV=user-local go run ./cmd/user-service

# Product service
APP_ENV=product-local go run ./cmd/product-service
```

### ❌ GORM auto-migration errors

**Cause:** Database schema conflicts from a previous run.

**Fix:** Drop and recreate the database:

```bash
psql -U postgres -c "DROP DATABASE auth_db;"
psql -U postgres -c "CREATE DATABASE auth_db;"
psql -U postgres -d auth_db -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
```

---

## Useful Makefile Commands

The project includes a comprehensive `Makefile` with 315 lines of commands. Here are the most useful ones:

### Build & Run

```bash
make build              # Build all 3 services → bin/ directory
make build-auth-service # Build only auth service
make run-auth-service   # Run auth service (go run)
make dev                # Run with hot reload (requires Air)
```

### Testing

```bash
make test               # Run all tests
make test-coverage      # Run tests with HTML coverage report
make test-integration   # Run integration tests
```

### Code Quality

```bash
make fmt                # Format all Go code
make lint               # Run golangci-lint
make lint-fix           # Auto-fix lint errors
make vet                # Run go vet (built-in static analysis)
make security           # Check for known vulnerabilities
```

### Docker

```bash
make docker-up          # Start all Docker containers
make docker-down        # Stop all Docker containers
make docker-logs        # Tail container logs
make docker-restart     # Restart all containers
```

### Generation

```bash
make wire               # Regenerate Wire dependency injection
make swagger            # Generate Swagger/OpenAPI docs (outputs to cmd/*/docs/)
make mocks              # Generate test mocks
```

### Dependencies

```bash
make deps               # Download dependencies + tidy
make update-deps        # Update all dependencies to latest
```

---

## How It All Connects: A Visual Summary

Here's the complete local development setup when everything is running:

```
YOUR TERMINAL WINDOWS:
┌─────────────────────────────────────────────────────────────────┐
│ Tab 1: Auth Service                                             │
│ $ APP_ENV=local go run ./cmd/auth-service                       │
│ → Listening on :3100                                            │
│ → Connected to PostgreSQL (auth_db) on localhost:5432           │
│ → Connected to Redis on localhost:6379                          │
├─────────────────────────────────────────────────────────────────┤
│ Tab 2: User Service                                             │
│ $ APP_ENV=user-local go run ./cmd/user-service                  │
│ → Listening on :3101                                            │
│ → Connected to PostgreSQL (user_db) on localhost:5432           │
│ → Connected to Redis on localhost:6379                          │
├─────────────────────────────────────────────────────────────────┤
│ Tab 3: Product Service                                          │
│ $ APP_ENV=product-local go run ./cmd/product-service            │
│ → Listening on :3102                                            │
│ → Connected to PostgreSQL (product_db) on localhost:5432        │
│ → Connected to Redis on localhost:6379                          │
└─────────────────────────────────────────────────────────────────┘

DOCKER CONTAINERS (background):
┌─────────────────────────────────────────────────────────────────┐
│ Redis             → localhost:6379  (Event Bus)                  │
│ Redis Insight     → localhost:5540  (Redis GUI)                  │
│ Prometheus        → localhost:9090  (Metrics Collection)         │
│ Grafana           → localhost:3000  (Dashboards, admin/admin)    │
│ Alertmanager      → localhost:9093  (Alert Management)           │
└─────────────────────────────────────────────────────────────────┘

LOCAL PostgreSQL:
┌─────────────────────────────────────────────────────────────────┐
│ PostgreSQL        → localhost:5432                               │
│   ├── auth_db     (auth-service tables)                         │
│   ├── user_db     (user-service tables)                         │
│   └── product_db  (product-service tables)                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Start Cheat Sheet

For when you come back tomorrow and need to start everything again:

```bash
# 1. Start infrastructure
docker compose -f deployments/docker-compose.yml up -d redis redis-insight
docker compose -f deployments/docker-compose.yml --profile monitoring up -d

# 2. Start services (each in a separate terminal)
APP_ENV=local go run ./cmd/auth-service
APP_ENV=user-local go run ./cmd/user-service
APP_ENV=product-local go run ./cmd/product-service

# 3. Verify everything is running
curl http://localhost:3100/health  # Auth
curl http://localhost:3101/health  # User
curl http://localhost:3102/health  # Product

# 4. Stop everything when done
# Ctrl+C in each service terminal
docker compose -f deployments/docker-compose.yml --profile monitoring down
```

---

> **📚 Next Steps:**
>
> - Explore the codebase starting from `cmd/auth-service/main.go` → follow the dependency injection chain
> - Read `docs/standardization/CODE_STYLE.md` for coding conventions
> - Read `docs/standardization/GORM_BEST_PRACTICES.md` for database patterns
> - Try adding Swagger annotations to handlers and running `make swagger`
