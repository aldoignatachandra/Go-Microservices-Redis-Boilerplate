# Comprehensive Code Review & Gap Analysis

## Go Microservices Redis Pub/Sub Boilerplate

**Date:** 2026-03-05  
**Reviewer:** AI Senior Backend Engineer (10+ years, expertise in Go & Node.js)  
**Scope:** Full codebase deep scan — 78 Go source files, infrastructure, docs, tests

---

## Executive Summary

Your boilerplate is architecturally sound and follows established patterns from Big Tech (Google Wire DI, Uber Zap logging, Netflix-style circuit breakers). The Clean Architecture layer separation is correctly enforced, and the Redis Streams event-driven communication is production-worthy.

However, after cross-referencing the [implementation plan](./2026-03-03-detailed-implementation-plan.md) with the actual code, I've identified **critical gaps** that need to be addressed before this can serve as a reliable learning boilerplate or production starter kit.

---

## 1. What's Working Well ✅

| Area                                  | Status              | Details                                                                                                                       |
| :------------------------------------ | :------------------ | :---------------------------------------------------------------------------------------------------------------------------- |
| **Clean Architecture**                | ✅ Excellent        | Strict `domain → repository → usecase → delivery` separation across all 3 services                                            |
| **EventBus (pkg/eventbus)**           | ✅ Production-ready | Full Redis Streams implementation with `XADD`, `XREADGROUP`, `XACK`, pending message claiming, retry with exponential backoff |
| **Config Management**                 | ✅ Solid            | Viper-based with `configs/local.yaml`, supports env overrides, fully typed structs                                            |
| **Middleware Registry**               | ✅ Complete         | RequestID, Logging, Recovery, CORS, Auth (JWT), RateLimit, Timeout, AdminOnly                                                 |
| **Graceful Shutdown**                 | ✅ Implemented      | Signal handling, context-based timeout, resource cleanup in auth & product services                                           |
| **Health Probes**                     | ✅ K8s-ready        | `/health`, `/ready`, `/live` endpoints with dependency checks                                                                 |
| **Prometheus Metrics**                | ✅ Functional       | HTTP request counters, duration histograms, Redis pub/sub counters                                                            |
| **Observability (pkg/observability)** | ✅ Present          | Alerting, logging, metrics, and tracing modules                                                                               |
| **Resilience**                        | ✅ Implemented      | Circuit breaker (Sony gobreaker) + retry with exponential backoff                                                             |
| **Makefile**                          | ✅ Comprehensive    | 315 lines, covering build, test, lint, Docker, K8s, CI, and Swagger                                                           |
| **Wire DI**                           | ✅ All 3 services   | `wire.go` + `wire_gen.go` present for auth, user, and product                                                                 |
| **Monitoring Stack**                  | ✅ Present          | Prometheus, Alertmanager, and Grafana dashboard configs                                                                       |

---

## 2. Critical Gaps Found 🚨

### 2.1 Empty `api/` Directory — No Swagger/OpenAPI Docs Generated

**Current state:** Both `api/openapi/` and `api/proto/` directories are **completely empty**.

**What's needed:**

- The Makefile has `make swagger` wired up to generate Swagger docs via `swag init`, but it has **never been executed**
- The handlers in `internal/*/delivery/handler.go` are **missing Swagger annotations** (`@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`)
- Without annotations, `swag init` will generate empty/useless docs

**Node.js analogy (for your context):**  
In Hono, you'd use `@hono/zod-openapi` to auto-generate docs from Zod schemas. In Go, the equivalent is adding comment annotations directly above handler functions, and then running `swag init` to parse them into OpenAPI JSON.

**Priority:** 🔴 HIGH — This is the most impactful learning gap. API documentation is essential for a boilerplate.

### 2.2 Test Coverage Is Extremely Low

**Current state:** Only **4 test files exist**, all within the User service:

| Test File                    | Type            | Location                  |
| :--------------------------- | :-------------- | :------------------------ |
| `handler_test.go`            | Unit (delivery) | `internal/user/delivery/` |
| `user_usecase_test.go`       | Unit (usecase)  | `internal/user/usecase/`  |
| `user_usecase_bench_test.go` | Benchmark       | `internal/user/usecase/`  |
| `user_integration_test.go`   | Integration     | `internal/user/`          |

**What's completely missing:**

- ❌ **Auth service tests** — No tests for Register, Login, RefreshToken, JWT validation
- ❌ **Product service tests** — No tests for CRUD operations, stock management
- ❌ **EventBus tests** — No tests for Producer/Consumer (critical infrastructure)
- ❌ **Middleware tests** — No tests for Auth, RateLimit, Recovery middleware
- ❌ **Config/Database tests** — No tests for connection handling, config loading
- ❌ **Integration test infrastructure** — `test/suite/suite.go` and `test/testutil/testutil.go` exist but no actual integration test suites for auth/product

**The implementation plan targets 80%+ coverage**, but the current codebase is likely below 10%.

**Priority:** 🔴 HIGH

### 2.3 Inconsistent Service Entry Points (`cmd/*/main.go`)

**Current state:** The 3 services have **different patterns** for their `main.go`:

| Service     | Pattern                               | Has App Struct                 | Uses logger pkg   | Graceful Shutdown Style      |
| :---------- | :------------------------------------ | :----------------------------- | :---------------- | :--------------------------- |
| **Auth**    | Full App struct + `setupHTTPServer()` | ✅                             | ✅                | `cfg.Server.ShutdownTimeout` |
| **Product** | Full App struct + `setupHTTPServer()` | ✅                             | ✅                | `cfg.Server.ShutdownTimeout` |
| **User**    | Minimal, different style              | ❌ Uses inline `app` from Wire | ❌ Uses `panic()` | Hardcoded `10*time.Second`   |

**Why this matters:** As a boilerplate, consistency is non-negotiable. A new developer looking at the user service will learn a different (and worse) pattern than auth/product.

**Specific Issues in user-service `main.go`:**

- Uses `panic()` for init errors instead of `log.Fatalf()` or `logger.Fatal()`
- Hardcodes `10*time.Second` shutdown timeout instead of using config
- Does not use the `pkg/logger` package at all
- Registers `gin.Recovery()` middleware **after** starting route handlers (order matters!)
- Does not set up Prometheus metrics
- Does not apply CORS middleware from the delivery package

**Priority:** 🔴 HIGH

### 2.4 Missing Per-Service YAML Configs

**Current state:** Only `configs/local.yaml` exists, hardcoded for `auth-service`.

**What's needed:**

- The implementation plan specifies `development.yaml`, `staging.yaml`, `production.yaml`
- More critically, there are NO separate configs for user-service (port 3101) and product-service (port 3102)
- Each service connects to a different DB (`auth_db`, `user_db`, `product_db`) but the single config only defines `auth_db`

**Impact:** You cannot run all 3 services simultaneously with the current config. They'd all try to bind to port 3100 and connect to `auth_db`.

**Priority:** 🔴 HIGH

### 2.5 Missing Dockerfiles

**Current state:**

- `deployments/docker/Dockerfile.auth` ✅ exists
- `deployments/docker/Dockerfile.base` ✅ exists
- `deployments/docker/Dockerfile.dev` ✅ exists
- `deployments/docker/Dockerfile.user` ❌ **missing**
- `deployments/docker/Dockerfile.product` ❌ **missing**

**The Makefile's `docker-build-prod` target expects** `Dockerfile.user` and `Dockerfile.product` to exist.

**Priority:** 🟡 MEDIUM

---

## 3. Code Quality Issues 🔍

### 3.1 Custom `itoa()` in `pkg/config/config.go`

```go
// itoa converts int to string without importing strconv.
func itoa(i int) string { ... }
```

This is a **hand-rolled integer-to-string converter** that reimplements `strconv.Itoa()`. There is absolutely no reason to avoid importing `strconv` — it's a stdlib package with zero overhead. This is a code smell that a code reviewer at Google would flag immediately.

**Better approach:**

```go
import "fmt"

func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}
```

### 3.2 Async Event Publishing Without Error Handling

In `internal/product/usecase/product_usecase.go`:

```go
// Publish asynchronously
go func() {
    _, _ = uc.eventBus.Publish(context.Background(), eventbus.StreamProductEvents, ebEvent)
}()
```

**Issues:**

1. Errors from `Publish()` are silently discarded (`_, _ =`)
2. Uses `context.Background()` instead of propagating the parent context
3. If the application shuts down, in-flight goroutines publishing events may be killed before finishing

**Recommendation:** At minimum, log the error. Better yet, use a buffered channel as an outbox queue that is drained gracefully on shutdown.

### 3.3 No `internal/common` Implementations

The `internal/common/` directory contains 3 subdirectories:

- `constants/` — empty or placeholder
- `errors/` — empty or placeholder
- `middleware/` — empty or placeholder

These were planned in the implementation plan but appear not to be implemented. The actual middleware lives in `pkg/middleware/` instead.

### 3.4 Missing Error Wrapping in Auth Service

In `internal/auth/delivery/middleware.go`, check if JWT validation errors are wrapped with `fmt.Errorf("...: %w", err)` for proper error chain propagation. This is critical for debugging.

---

## 4. Architecture Comparison: Plan vs Reality

| Implementation Plan Item                         | Planned     | Actual Status     |
| :----------------------------------------------- | :---------- | :---------------- |
| Project initialization                           | ✅          | ✅ Done           |
| Docker Compose                                   | ✅          | ✅ Done           |
| `.air.toml` hot reload                           | ✅          | ✅ Done           |
| `.golangci.yml` linting                          | ✅          | ✅ Done           |
| `pkg/logger` (Zap)                               | ✅          | ✅ Done           |
| `pkg/config` (Viper)                             | ✅          | ✅ Done           |
| `pkg/database/postgres`                          | ✅          | ✅ Done           |
| `pkg/database/redis`                             | ✅          | ✅ Done           |
| `pkg/eventbus` (Redis Streams)                   | ✅          | ✅ Done           |
| `pkg/resilience` (Circuit Breaker + Retry)       | ✅          | ✅ Done           |
| `pkg/server` (Graceful + Health)                 | ✅          | ✅ Done           |
| `pkg/utils` (Hash, JWT, Response)                | ✅          | ✅ Done           |
| `pkg/metrics` (Prometheus)                       | ✅          | ✅ Done           |
| `pkg/middleware` (Full set)                      | ✅          | ✅ Done           |
| Auth Service domain/dto/repo/usecase/delivery    | ✅          | ✅ Done           |
| User Service domain/dto/repo/usecase/delivery    | ✅          | ✅ Done           |
| Product Service domain/dto/repo/usecase/delivery | ✅          | ✅ Done           |
| Wire DI (all 3 services)                         | ✅          | ✅ Done           |
| Swagger/OpenAPI annotations                      | ✅          | ❌ **Missing**    |
| Swagger doc generation (`api/openapi/`)          | ✅          | ❌ **Empty**      |
| Per-service YAML configs                         | ✅          | ❌ **Only auth**  |
| Docker images (user, product)                    | ✅          | ❌ **Missing**    |
| Unit tests (auth, product)                       | ✅          | ❌ **Missing**    |
| Integration tests                                | ✅          | ⚠️ **Only user**  |
| `internal/common/` utilities                     | ✅          | ❌ **Empty dirs** |
| OpenTelemetry tracing                            | ⬜ Optional | ⚠️ Stub exists    |
| `proto/` (gRPC definitions)                      | ⬜ Future   | ❌ Empty          |

---

## 5. Recommended Action Items (Prioritized)

### Phase A: Critical Fixes (Must Do)

| #   | Item                                                                                                                   | Impact              | Effort         |
| :-- | :--------------------------------------------------------------------------------------------------------------------- | :------------------ | :------------- |
| A1  | Create per-service YAML configs (`user-local.yaml`, `product-local.yaml`) with correct ports (3101, 3102) and DB names | 🔴 Blocks running   | Low            |
| A2  | Fix `cmd/user-service/main.go` to match auth/product pattern (App struct, logger, configurable shutdown)               | 🔴 Inconsistency    | Medium         |
| A3  | Add Swagger annotations to all handlers across 3 services                                                              | 🔴 Missing API docs | High           |
| A4  | Run `make swagger` to populate `api/openapi/`                                                                          | 🔴 Empty dir        | Low (after A3) |
| A5  | Replace custom `itoa()` with `strconv.Itoa()` or `fmt.Sprintf()`                                                       | 🟡 Code smell       | Low            |

### Phase B: Test Coverage (Should Do)

| #   | Item                                                                | Impact              | Effort |
| :-- | :------------------------------------------------------------------ | :------------------ | :----- |
| B1  | Add auth usecase unit tests (Register, Login, RefreshToken, Logout) | 🔴 No auth tests    | Medium |
| B2  | Add product usecase unit tests (CRUD, Stock, Events)                | 🔴 No product tests | Medium |
| B3  | Add eventbus unit tests (Producer, Consumer, pending claim)         | 🔴 Critical infra   | Medium |
| B4  | Add middleware unit tests (Auth, RateLimit)                         | 🟡 Good practice    | Medium |
| B5  | Add auth integration tests                                          | 🟡 Completeness     | High   |

### Phase C: Infrastructure (Nice to Have)

| #   | Item                                                                | Impact                | Effort |
| :-- | :------------------------------------------------------------------ | :-------------------- | :----- |
| C1  | Create `Dockerfile.user` and `Dockerfile.product`                   | 🟡 Can't containerize | Low    |
| C2  | Add environment-specific configs (development, staging, production) | 🟡 DevOps ready       | Low    |
| C3  | Implement `internal/common/` or remove empty directories            | 🟡 Clean structure    | Low    |
| C4  | Add error logging in async event publishing goroutines              | 🟡 Observability      | Low    |
| C5  | Add CI/CD GitHub Actions workflow                                   | 🟡 Automation         | Medium |

---

## 6. Comparison: Node.js (Hono) vs Go Pattern Mapping

Since you're transitioning from Node.js/TypeScript, here's a direct mapping of concepts:

| Node.js Concept          | Go Equivalent in This Project                             | Key File(s)                    |
| :----------------------- | :-------------------------------------------------------- | :----------------------------- |
| `class UserService`      | `type userUseCase struct` + `UserUseCase` interface       | `internal/*/usecase/`          |
| Zod schemas              | Go struct tags (`binding:"required"`, `validate:"email"`) | `internal/*/dto/request.go`    |
| Express middleware       | `gin.HandlerFunc`                                         | `pkg/middleware/`              |
| `process.on('SIGTERM')`  | `signal.Notify(quit, syscall.SIGTERM)`                    | `cmd/*/main.go`                |
| `ioredis` streams        | `go-redis/v9` `XADD`/`XREADGROUP`                         | `pkg/eventbus/`                |
| `drizzle-orm` queries    | `gorm.io/gorm` `db.Where().First()`                       | `internal/*/repository/`       |
| `pino` JSON logger       | `go.uber.org/zap` structured logger                       | `pkg/logger/`                  |
| `npm run dev` (nodemon)  | `make dev` (Air hot reload)                               | `.air.toml`, `Makefile`        |
| `TypeDI` / `tsyringe`    | `google/wire` compile-time DI                             | `cmd/*/wire.go`                |
| `jest.mock()`            | `testify/mock` + `mockery` generated mocks                | `internal/user/usecase/mocks/` |
| `Promise.all()`          | `sync.WaitGroup` + goroutines                             | Go concurrency patterns        |
| `try/catch`              | `if err != nil { return fmt.Errorf("...: %w", err) }`     | Everywhere                     |
| `interface` (TypeScript) | `type X interface {}` (implicitly satisfied)              | Domain layer interfaces        |
| `export class`           | Uppercase first letter (`func NewApp()` = exported)       | Go naming convention           |
| `private method`         | Lowercase first letter (`func (uc *useCase) validate()`)  | Go naming convention           |

---

## 7. Learning Path Recommendation

As a backend engineer moving from Node.js to Go, I recommend tackling the action items in this order for maximum learning:

1. **Start with A1** (per-service configs) — You'll learn Viper config loading patterns
2. **Then A2** (fix user-service main.go) — Teaches you Go's `struct` + interface patterns and graceful shutdown
3. **Then B1** (auth tests) — Table-driven testing is THE fundamental Go testing pattern
4. **Then A3** (Swagger annotations) — Bridges your Hono OpenAPI knowledge to Go's annotation system
5. **Then B3** (eventbus tests) — Deepens Redis Streams understanding + Go concurrency testing

---

## Next Steps

I recommend we work through these items iteratively. Pick a phase (A, B, or C) and I'll create a detailed implementation plan for each specific task with code examples.

> **Note:** This document is intended to be a living review. As we fix items, I'll update the status in this table so you can track progress.
