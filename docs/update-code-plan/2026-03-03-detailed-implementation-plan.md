# Comprehensive Implementation Plan: Go Microservices Architecture

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a high-performance, production-grade Go microservices boilerplate with Redis Pub/Sub for event-driven communication, following Big Tech standards (Uber, Meta, Google).

**Architecture:** Monorepo with three independent services (auth, user, product) communicating via Redis Streams. Each service follows Clean Architecture with strict layer separation (delivery → usecase → repository → domain). Services share a common "platform kit" (pkg/) for logging, config, and infrastructure.

**Tech Stack:** Go 1.25+, Gin (HTTP), GORM (ORM), go-redis v9, Zap (logging), Viper (config), Wire (DI), Sony Gobreaker, Swag (OpenAPI), Testify (testing)

---

## Ultra-Think Analysis: Problem Space

### Core Challenge

Port a JavaScript microservices boilerplate to Go while:

1. **Maintaining feature parity** with the Bun/Hono versions
2. **Achieving production-grade quality** suitable for enterprise deployment
3. **Providing a learning path** for developers new to Go

### Key Constraints

| Constraint             | Impact                                                      |
| ---------------------- | ----------------------------------------------------------- |
| User is Go newbie      | Need extensive comments, idiomatic examples, clear patterns |
| Must match JS features | Rate limiting, health checks, metrics, graceful shutdown    |
| Production-ready       | Circuit breakers, connection pooling, structured logging    |
| Kubernetes-ready       | Health probes, graceful shutdown, resource limits           |

### Critical Success Factors

1. **All blocking operations use context.Context** (Go idiom)
2. **Every error is wrapped with context** (`fmt.Errorf("operation failed: %w", err)`)
3. **All tests use table-driven pattern** with subtests
4. **Graceful shutdown** handles both HTTP server and Redis consumers
5. **Structured JSON logging** with trace ID correlation

---

## 1. Technology Stack (Production-Grade)

### Core Stack Comparison

| Component           | JavaScript (Reference) | Go (Selected)               | Rationale / Big Tech Usage                                                                |
| :------------------ | :--------------------- | :-------------------------- | :---------------------------------------------------------------------------------------- |
| **Language**        | TypeScript (Bun)       | **Go 1.25+**                | Latest stable with improved generics and performance                                      |
| **Web Framework**   | Hono                   | **Gin**                     | `github.com/gin-gonic/gin` - 80k+ stars, Express-like API, extensive middleware ecosystem |
| **ORM**             | Drizzle                | **GORM**                    | `gorm.io/gorm` - Dynamic, no code-gen required, auto-migrations, soft delete support      |
| **Validation**      | Zod                    | **go-playground/validator** | Industry standard struct-tag validation (built into Gin)                                  |
| **Redis Client**    | ioredis                | **go-redis v9**             | `github.com/redis/go-redis/v9` - Supports Sentinel/Cluster, type-safe                     |
| **Circuit Breaker** | Opossum                | **Sony Gobreaker**          | `github.com/sony/gobreaker` - Battle-tested at Sony, simple API                           |
| **Rate Limiter**    | Custom                 | **gin-contrib/limiter**     | Redis-backed distributed rate limiting                                                    |
| **Logging**         | Pino                   | **Zap**                     | `go.uber.org/zap` - Uber's zero-allocation logger, structured JSON                        |
| **Config**          | Custom (Zod)           | **Viper**                   | `github.com/spf13/viper` - 12-factor app config with env overrides                        |
| **DI**              | TypeDI                 | **Wire**                    | `github.com/google/wire` - Compile-time DI, catches errors at build time                  |
| **Swagger**         | Hono OpenAPI           | **Swag**                    | `github.com/swaggo/swag` - Generates docs from annotations                                |
| **Testing**         | Bun Test               | **Testify**                 | `github.com/stretchr/testify` - Assertions + mocking, table-driven tests                  |
| **Metrics**         | Prometheus (optional)  | **Prometheus client**       | `github.com/prometheus/client_golang` - Direct integration                                |
| **Tracing**         | None                   | **OpenTelemetry**           | `go.opentelemetry.io/otel` - Distributed tracing for microservices                        |

### Additional Production Dependencies

```
# UUID generation
github.com/google/uuid

# Time utilities
github.com/jinzhu/now

# Environment variables
github.com/joho/godotenv

# HTTP client with retry
github.com/go-resty/resty/v2

# Secure password hashing
golang.org/x/crypto/bcrypt

# JWT
github.com/golang-jwt/jwt/v5

# Middleware
github.com/gin-contrib/cors
github.com/gin-contrib/requestid
github.com/gin-contrib/recovery
```

---

## 2. Project Structure (Standard Go Layout + DDD)

```
go-microservices-redis-pubsub-boilerplate/
├── cmd/                              # MAIN ENTRY POINTS (One per service)
│   ├── auth-service/
│   │   ├── main.go                   # Entry point - DI wiring only
│   │   └── wire.go                   # Wire dependency injection
│   ├── user-service/
│   │   ├── main.go
│   │   └── wire.go
│   └── product-service/
│       ├── main.go
│       └── wire.go
│
├── internal/                         # PRIVATE BUSINESS LOGIC
│   ├── auth/                         # Auth Bounded Context
│   │   ├── domain/                   # Entities, value objects
│   │   │   ├── user.go
│   │   │   ├── session.go
│   │   │   └── events.go
│   │   ├── dto/                      # Data Transfer Objects
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   ├── repository/               # Data access interfaces
│   │   │   └── user_repository.go
│   │   ├── usecase/                  # Business logic
│   │   │   ├── auth_usecase.go
│   │   │   └── session_usecase.go
│   │   └── delivery/                 # HTTP handlers
│   │       ├── handler.go
│   │       ├── routes.go
│   │       └── middleware.go
│   ├── user/                         # User Bounded Context
│   │   ├── domain/
│   │   ├── dto/
│   │   ├── repository/
│   │   ├── usecase/
│   │   └── delivery/
│   ├── product/                      # Product Bounded Context
│   │   ├── domain/
│   │   ├── dto/
│   │   ├── repository/
│   │   ├── usecase/
│   │   └── delivery/
│   └── common/                       # Shared internal utilities
│       ├── middleware/               # Cross-cutting middleware
│       │   ├── auth.go
│       │   ├── ratelimit.go
│       │   ├── logger.go
│       │   ├── cors.go
│       │   └── recovery.go
│       ├── errors/                   # Custom error types
│       │   └── errors.go
│       └── constants/                # Shared constants
│           └── events.go
│
├── pkg/                              # PUBLIC SHARED LIBRARIES ("Platform Kit")
│   ├── logger/                       # Structured logging (Zap)
│   │   ├── logger.go
│   │   └── context.go                # Context-aware logging
│   ├── config/                       # Configuration (Viper)
│   │   ├── config.go
│   │   └── loader.go
│   ├── database/                     # Database connections
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── eventbus/                     # Redis Streams abstraction
│   │   ├── eventbus.go               # Interface
│   │   ├── producer.go
│   │   ├── consumer.go
│   │   └── options.go
│   ├── resilience/                   # Resilience patterns
│   │   ├── circuit_breaker.go
│   │   └── retry.go
│   ├── server/                       # HTTP server utilities
│   │   ├── server.go
│   │   ├── graceful.go
│   │   └── health.go
│   ├── metrics/                      # Prometheus metrics
│   │   └── metrics.go
│   └── utils/                        # Common utilities
│       ├── hash.go
│       ├── jwt.go
│       └── response.go
│
├── api/                              # API Definitions
│   ├── openapi/                      # OpenAPI specs
│   └── proto/                        # Protocol buffers (future gRPC)
│
├── configs/                          # Configuration files
│   ├── local.yaml
│   ├── development.yaml
│   ├── staging.yaml
│   └── production.yaml
│
├── deployments/                      # INFRASTRUCTURE
│   ├── docker/
│   │   ├── Dockerfile.auth
│   │   ├── Dockerfile.user
│   │   ├── Dockerfile.product
│   │   └── Dockerfile.base
│   ├── docker-compose.yml            # Local development
│   ├── docker-compose.prod.yml       # Production stack
│   └── k8s/                          # Kubernetes manifests
│       ├── base/
│       │   ├── namespace.yaml
│       │   ├── configmap.yaml
│       │   └── secrets.yaml
│       └── overlays/
│           ├── local/
│           ├── staging/
│           └── production/
│
├── scripts/                          # Build and utility scripts
│   ├── build.sh
│   ├── migrate.sh
│   └── seed.sh
│
├── test/                             # Integration tests
│   ├── integration/
│   └── e2e/
│
├── docs/                             # Documentation
│   ├── architecture/
│   ├── api/
│   └── runbook/
│
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml                     # Linter configuration
├── .air.toml                         # Hot reload configuration
└── README.md
```

### Why This Structure?

| Directory      | Purpose                                    | Big Tech Pattern      |
| -------------- | ------------------------------------------ | --------------------- |
| `cmd/`         | Entry points only - NO business logic      | Google, Uber          |
| `internal/`    | Private code - compiler enforced           | Go standard           |
| `pkg/`         | Shared libraries - platform team maintains | Netflix, Uber         |
| `configs/`     | Environment-specific configuration         | 12-Factor App         |
| `deployments/` | Docker + K8s manifests                     | Cloud-native standard |

---

## 3. Architecture Patterns

### A. Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                      DELIVERY LAYER (HTTP)                      │
│  Gin handlers, middleware, routes, request/response binding     │
├─────────────────────────────────────────────────────────────────┤
│                      USECASE LAYER (Business Logic)             │
│  Application services, orchestration, domain events             │
├─────────────────────────────────────────────────────────────────┤
│                      REPOSITORY LAYER (Data Access)             │
│  Database operations, external API calls, cache access          │
├─────────────────────────────────────────────────────────────────┤
│                      DOMAIN LAYER (Core)                        │
│  Entities, value objects, business rules, domain events         │
└─────────────────────────────────────────────────────────────────┘
```

### B. CQRS Pattern Implementation

```go
// internal/product/usecase/product_usecase.go

type ProductUseCase interface {
    // ═══════════════════════════════════════════════════════════
    // COMMANDS (Write Operations - Mutate State)
    // ═══════════════════════════════════════════════════════════
    CreateProduct(ctx context.Context, cmd *dto.CreateProductCommand) (*dto.ProductResponse, error)
    UpdateProduct(ctx context.Context, id uuid.UUID, cmd *dto.UpdateProductCommand) error
    DeleteProduct(ctx context.Context, id uuid.UUID) error
    RestoreProduct(ctx context.Context, id uuid.UUID) error

    // ═══════════════════════════════════════════════════════════
    // QUERIES (Read Operations - No Side Effects)
    // ═══════════════════════════════════════════════════════════
    GetProduct(ctx context.Context, query *dto.GetProductQuery) (*dto.ProductResponse, error)
    ListProducts(ctx context.Context, query *dto.ListProductsQuery) (*dto.ProductListResponse, error)
    GetProductBySKU(ctx context.Context, sku string) (*dto.ProductResponse, error)
}
```

### C. Event-Driven Communication (Redis Streams)

```
┌─────────────────┐    UserCreated     ┌─────────────────┐
│  AUTH SERVICE   │ ─────────────────► │  USER SERVICE   │
│  (Publisher)    │                    │  (Consumer)     │
└─────────────────┘                    └─────────────────┘
         │                                     │
         │ ProductCreated                      │ ActivityLogged
         ▼                                     ▼
┌─────────────────┐                    ┌─────────────────┐
│ PRODUCT SERVICE │                    │  USER SERVICE   │
│  (Publisher)    │                    │  (Consumer)     │
└─────────────────┘                    └─────────────────┘

Redis Streams:
├── auth:events     → login, logout, register
├── users:events    → user.created, user.updated, user.deleted
├── products:events → product.created, product.updated, product.deleted
└── activity:log    → audit trail (all events)
```

### D. Redis Streams Interface

```go
// pkg/eventbus/eventbus.go

package eventbus

import "context"

// EventBus defines the contract for pub/sub operations
type EventBus interface {
    // Publish sends an event to a Redis stream
    Publish(ctx context.Context, stream string, event Event) error

    // Subscribe consumes events from a stream using consumer groups
    Subscribe(ctx context.Context, opts SubscribeOptions) error

    // Ack acknowledges successful processing of a message
    Ack(ctx context.Context, stream, group, id string) error

    // Close gracefully shuts down the event bus
    Close() error
}

// Event represents a domain event
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Source    string                 `json:"source"`
    Timestamp int64                  `json:"timestamp"`
    Payload   map[string]interface{} `json:"payload"`
    Metadata  map[string]string      `json:"metadata,omitempty"`
}

// SubscribeOptions configures the consumer
type SubscribeOptions struct {
    Stream      string
    Group       string
    Consumer    string
    BatchSize   int64
    BlockMs     int64
    Handler     func(ctx context.Context, event Event) error
    ErrorHandler func(ctx context.Context, event Event, err error)
}
```

---

## 4. Production-Grade Features

### A. Graceful Shutdown (Critical for Kubernetes)

```go
// pkg/server/graceful.go

package server

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// GracefulServer wraps http.Server with graceful shutdown
type GracefulServer struct {
    *http.Server
    shutdownTimeout time.Duration
    onShutdown      []func(context.Context) error
}

// WaitForShutdown blocks until SIGINT/SIGTERM, then gracefully shuts down
func (s *GracefulServer) WaitForShutdown() {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
    defer cancel()

    // Run custom shutdown hooks (close DB, Redis consumers, etc.)
    for _, hook := range s.onShutdown {
        if err := hook(ctx); err != nil {
            log.Printf("Shutdown hook error: %v", err)
        }
    }

    if err := s.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }

    log.Println("Server stopped")
}
```

### B. Health Check Endpoints (Kubernetes Probes)

```go
// internal/common/delivery/health.go

package delivery

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints
type HealthHandler struct {
    db     HealthChecker
    redis  HealthChecker
}

// HealthChecker interface for dependency health checks
type HealthChecker interface {
    Ping(ctx context.Context) error
}

// @Summary Public health check
// @Description Returns 200 if service is running (for load balancers)
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *HealthHandler) PublicHealth(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "service": c.GetString("service_name"),
    })
}

// @Summary Admin health check
// @Description Detailed health with dependency status (requires system auth)
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /admin/health [get]
// @Security SystemAuth
func (h *HealthHandler) AdminHealth(c *gin.Context) {
    ctx := c.Request.Context()
    status := http.StatusOK

    checks := map[string]string{
        "database": "ok",
        "redis":    "ok",
    }

    if err := h.db.Ping(ctx); err != nil {
        checks["database"] = err.Error()
        status = http.StatusServiceUnavailable
    }

    if err := h.redis.Ping(ctx); err != nil {
        checks["redis"] = err.Error()
        status = http.StatusServiceUnavailable
    }

    c.JSON(status, gin.H{
        "status":  "ok",
        "service": c.GetString("service_name"),
        "checks":  checks,
    })
}

// ReadyProbe for Kubernetes readiness probe
func (h *HealthHandler) ReadyProbe(c *gin.Context) {
    ctx := c.Request.Context()

    if err := h.db.Ping(ctx); err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false})
        return
    }

    c.JSON(http.StatusOK, gin.H{"ready": true})
}

// LiveProbe for Kubernetes liveness probe
func (h *HealthHandler) LiveProbe(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"alive": true})
}
```

### C. Structured Logging with Context

```go
// pkg/logger/context.go

package logger

import (
    "context"

    "go.uber.org/zap"
)

// contextKey for logger in context
type contextKey string

const (
    RequestIDKey contextKey = "request_id"
    UserIDKey    contextKey = "user_id"
    TraceIDKey   contextKey = "trace_id"
)

// WithContext creates a logger with context values
func WithContext(ctx context.Context, base *zap.Logger) *zap.Logger {
    fields := []zap.Field{}

    if requestID := ctx.Value(RequestIDKey); requestID != nil {
        fields = append(fields, zap.String("request_id", requestID.(string)))
    }

    if userID := ctx.Value(UserIDKey); userID != nil {
        fields = append(fields, zap.String("user_id", userID.(string)))
    }

    if traceID := ctx.Value(TraceIDKey); traceID != nil {
        fields = append(fields, zap.String("trace_id", traceID.(string)))
    }

    return base.With(fields...)
}

// Usage in handlers:
// logger.WithContext(c.Request.Context(), log).Info("user action", zap.String("action", "login"))
```

### D. Rate Limiting Middleware

```go
// internal/common/middleware/ratelimit.go

package middleware

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ulule/limiter/v3"
    mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/redis"
)

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
    Enabled    bool
    RedisAddr  string
    Requests   int           // Max requests
    Duration   time.Duration // Per duration
    KeyFunc    func(c *gin.Context) string
}

// NewRateLimiter creates a rate limiting middleware
func NewRateLimiter(cfg RateLimitConfig) (gin.HandlerFunc, error) {
    if !cfg.Enabled {
        return func(c *gin.Context) { c.Next() }, nil
    }

    store, err := redis.NewStoreWithOptions(&redis.StoreOptions{
        Address: cfg.RedisAddr,
    })
    if err != nil {
        return nil, err
    }

    rate := limiter.Rate{
        Period: cfg.Duration,
        Limit:  int64(cfg.Requests),
    }

    instance := limiter.New(store, rate, limiter.WithTrustForwardHeader(true))

    middleware := mgin.NewMiddleware(instance, mgin.WithKeyGetter(cfg.KeyFunc))

    return func(c *gin.Context) {
        middleware(c)

        if c.IsAborted() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "success": false,
                "message": "rate limit exceeded",
                "error": gin.H{
                    "code": "RATE_LIMIT_EXCEEDED",
                },
            })
            c.Abort()
            return
        }
        c.Next()
    }, nil
}
```

### E. Circuit Breaker Pattern

```go
// pkg/resilience/circuit_breaker.go

package resilience

import (
    "time"

    "github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds configuration
type CircuitBreakerConfig struct {
    Name          string
    MaxRequests   uint32
    Timeout       time.Duration
    Interval      time.Duration
    FailureRatio  float64
    MinRequests   uint32
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker {
    return gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        cfg.Name,
        MaxRequests: cfg.MaxRequests,
        Interval:    cfg.Interval,
        Timeout:     cfg.Timeout,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= cfg.MinRequests && failureRatio >= cfg.FailureRatio
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            // Log state changes for observability
            // logger.Info("circuit breaker state changed",
            //     zap.String("name", name),
            //     zap.String("from", from.String()),
            //     zap.String("to", to.String()),
            // )
        },
    })
}

// Default configs for different dependencies
var (
    DatabaseBreakerConfig = CircuitBreakerConfig{
        Name:         "database",
        MaxRequests:  5,
        Timeout:      30 * time.Second,
        Interval:     60 * time.Second,
        FailureRatio: 0.6,
        MinRequests:  3,
    }

    RedisBreakerConfig = CircuitBreakerConfig{
        Name:         "redis",
        MaxRequests:  10,
        Timeout:      15 * time.Second,
        Interval:     30 * time.Second,
        FailureRatio: 0.5,
        MinRequests:  5,
    }
)
```

### F. Prometheus Metrics

```go
// pkg/metrics/metrics.go

package metrics

import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // HTTP metrics
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"service", "method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"service", "method", "path"},
    )

    // Redis metrics
    redisPubTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "redis_publish_total",
            Help: "Total events published to Redis",
        },
        []string{"service", "stream"},
    )

    redisSubTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "redis_consume_total",
            Help: "Total events consumed from Redis",
        },
        []string{"service", "stream", "status"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(redisPubTotal)
    prometheus.MustRegister(redisSubTotal)
}

// PrometheusHandler returns the Prometheus HTTP handler
func PrometheusHandler() gin.HandlerFunc {
    h := promhttp.Handler()
    return func(c *gin.Context) {
        h.ServeHTTP(c.Writer, c.Request)
    }
}

// MetricsMiddleware records HTTP metrics
func MetricsMiddleware(serviceName string) gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.FullPath()
        if path == "" {
            path = c.Request.URL.Path
        }

        timer := prometheus.NewTimer(
            httpRequestDuration.WithLabelValues(serviceName, c.Request.Method, path),
        )
        defer timer.ObserveDuration()

        c.Next()

        status := string(rune(c.Writer.Status()))
        httpRequestsTotal.WithLabelValues(serviceName, c.Request.Method, path, status).Inc()
    }
}
```

---

## 5. Implementation Roadmap

### Phase 1: Foundation & Infrastructure Setup (Week 1)

#### Task 1.1: Project Initialization

- [ ] Initialize `go.mod` with module name `github.com/yourorg/go-microservices-boilerplate`
- [ ] Set Go version to 1.25 in `go.mod`
- [ ] Create directory structure as defined in Section 2
- [ ] Initialize Git repository with `.gitignore` for Go
- [ ] Create `Makefile` with standard targets

#### Task 1.2: Development Environment

- [ ] Create `docker-compose.yml` with PostgreSQL, Redis, and optional Redis Insight
- [ ] Create `.air.toml` for hot reload configuration
- [ ] Create `.golangci.yml` with production linter rules
- [ ] Create `configs/local.yaml` with development defaults

#### Task 1.3: Core Infrastructure (pkg/)

- [ ] Implement `pkg/logger/` with Zap + context support
- [ ] Implement `pkg/config/` with Viper + environment variable support
- [ ] Implement `pkg/database/postgres.go` with connection pooling
- [ ] Implement `pkg/database/redis.go` with Sentinel-ready client

#### Task 1.4: Docker Setup

- [ ] Create `deployments/docker/Dockerfile.base` (multi-stage base)
- [ ] Create service-specific Dockerfiles
- [ ] Test local build: `docker-compose build`

### Phase 2: Shared Libraries (pkg/) (Week 1-2)

#### Task 2.1: EventBus Implementation

- [ ] Define `pkg/eventbus/eventbus.go` interface
- [ ] Implement `pkg/eventbus/producer.go` with XADD
- [ ] Implement `pkg/eventbus/consumer.go` with XREADGROUP
- [ ] Add error handling and retry logic
- [ ] Write table-driven tests

#### Task 2.2: Resilience Patterns

- [ ] Implement `pkg/resilience/circuit_breaker.go`
- [ ] Implement `pkg/resilience/retry.go` with exponential backoff
- [ ] Write tests with simulated failures

#### Task 2.3: Server Utilities

- [ ] Implement `pkg/server/server.go` with Gin setup
- [ ] Implement `pkg/server/graceful.go` for graceful shutdown
- [ ] Implement `pkg/server/health.go` with health check helpers

#### Task 2.4: Common Utilities

- [ ] Implement `pkg/utils/response.go` with standard API response
- [ ] Implement `pkg/utils/hash.go` with bcrypt
- [ ] Implement `pkg/utils/jwt.go` with JWT helpers

### Phase 3: Auth Service (Week 2-3)

#### Task 3.1: Domain Layer

- [ ] Create `internal/auth/domain/user.go` entity
- [ ] Create `internal/auth/domain/session.go` entity
- [ ] Create `internal/auth/domain/events.go` with event types

#### Task 3.2: DTO Layer

- [ ] Create `internal/auth/dto/request.go` with validation tags
- [ ] Create `internal/auth/dto/response.go`

#### Task 3.3: Repository Layer

- [ ] Define `internal/auth/repository/user_repository.go` interface
- [ ] Implement with GORM + circuit breaker
- [ ] Write repository tests with test containers

#### Task 3.4: Usecase Layer

- [ ] Implement `internal/auth/usecase/auth_usecase.go`
- [ ] Add JWT generation and validation
- [ ] Add session management
- [ ] Write usecase tests with mocked repository

#### Task 3.5: Delivery Layer

- [ ] Create `internal/auth/delivery/handler.go`
- [ ] Create `internal/auth/delivery/routes.go`
- [ ] Add Swagger annotations
- [ ] Generate docs: `swag init`

#### Task 3.6: Wire DI

- [ ] Create `cmd/auth-service/wire.go`
- [ ] Generate wire: `wire gen ./cmd/auth-service`

#### Task 3.7: Main Entry

- [ ] Create `cmd/auth-service/main.go`
- [ ] Wire all components
- [ ] Start HTTP server + event consumers

### Phase 4: User Service (Week 3-4)

#### Task 4.1: Domain Layer

- [ ] Create `internal/user/domain/user.go` entity
- [ ] Create `internal/user/domain/activity_log.go` entity

#### Task 4.2: DTO Layer

- [ ] Create `internal/user/dto/` with request/response

#### Task 4.3: Repository Layer

- [ ] Implement `internal/user/repository/user_repository.go`
- [ ] Implement `internal/user/repository/activity_repository.go`

#### Task 4.4: Usecase Layer

- [ ] Implement `internal/user/usecase/user_usecase.go`
- [ ] Implement activity logging consumer

#### Task 4.5: Delivery Layer

- [ ] Create handlers and routes
- [ ] Add Swagger annotations

#### Task 4.6: Event Consumer

- [ ] Subscribe to `auth:events`
- [ ] Subscribe to `products:events`
- [ ] Log all activities

### Phase 5: Product Service (Week 4-5)

#### Task 5.1-5.6: Same structure as User Service

- [ ] Domain, DTO, Repository, Usecase, Delivery, Wire

#### Task 5.7: Event Publisher

- [ ] Publish `product.created`, `product.updated`, `product.deleted`

### Phase 6: Common Middleware (Week 5)

#### Task 6.1: Authentication Middleware

- [ ] JWT validation
- [ ] Role-based access control

#### Task 6.2: Rate Limiting

- [ ] Redis-backed rate limiter
- [ ] Per-endpoint configuration

#### Task 6.3: Request ID & Logging

- [ ] Request ID generation
- [ ] Structured request logging

#### Task 6.4: CORS & Recovery

- [ ] CORS configuration
- [ ] Panic recovery middleware

### Phase 7: Observability (Week 5-6)

#### Task 7.1: Metrics

- [ ] Prometheus metrics endpoint
- [ ] HTTP request metrics
- [ ] Redis event metrics

#### Task 7.2: Tracing (Optional)

- [ ] OpenTelemetry setup
- [ ] Trace propagation

### Phase 8: Testing & Documentation (Week 6)

#### Task 8.1: Unit Tests

- [ ] 80%+ coverage target
- [ ] Table-driven tests
- [ ] Race detector: `go test -race ./...`

#### Task 8.2: Integration Tests

- [ ] Test containers for PostgreSQL
- [ ] Test containers for Redis
- [ ] API integration tests

#### Task 8.3: Documentation

- [ ] API documentation (Swagger)
- [ ] Architecture decision records
- [ ] Runbook for operations

---

## 6. Configuration Schema

```yaml
# configs/local.yaml

app:
  name: "auth-service"
  version: "1.0.0"
  env: "local"

server:
  host: "0.0.0.0"
  port: 3100
  read_timeout: "10s"
  write_timeout: "10s"
  shutdown_timeout: "10s"

database:
  host: "localhost"
  port: 5432
  name: "auth_db"
  user: "postgres"
  password: "postgres"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "5m"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 10

streams:
  max_len: 10000
  block_ms: 5000
  batch_size: 10
  consumer_group: "auth-service"
  consumer_name: "auth-1"

auth:
  jwt:
    secret: "your-super-secret-key-change-in-production"
    expires_in: "24h"
    refresh_expires_in: "168h"
  bcrypt:
    cost: 12

rate_limit:
  enabled: true
  requests: 100
  duration: "1m"

logging:
  level: "debug"
  format: "console" # "json" for production

metrics:
  enabled: true
  path: "/metrics"

tracing:
  enabled: false
  endpoint: "http://localhost:4317"

services:
  user_service: "http://localhost:3101"
  product_service: "http://localhost:3102"
```

---

## 7. Kubernetes Deployment

### Liveness, Readiness, Startup Probes

```yaml
# deployments/k8s/base/deployment.yaml

livenessProbe:
  httpGet:
    path: /live
    port: 3100
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 3100
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /live
    port: 3100
  initialDelaySeconds: 0
  periodSeconds: 1
  failureThreshold: 30
```

---

## 8. Makefile Commands

```makefile
.PHONY: all build test lint run clean deps docker-up docker-down

# Variables
GO := go
BINARY := auth-service
BUILD_DIR := bin
SERVICES := auth-service user-service product-service

# All services
all: deps build

# Build all services
build:
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		$(GO) build -o $(BUILD_DIR)/$$service ./cmd/$$service; \
	done

# Run tests with race detector
test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	$(GO) tool cover -html=coverage.out

# Run linters
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

# Run service (default: auth)
run:
	$(GO) run ./cmd/auth-service

# Run with hot reload
dev:
	air -c .air.toml

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Generate Wire DI
wire:
	wire gen ./cmd/...

# Generate Swagger docs
swagger:
	swag init -g cmd/auth-service/main.go -o api/openapi

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

# Migrations
migrate-up:
	$(GO) run ./cmd/migrate up

migrate-down:
	$(GO) run ./cmd/migrate down

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build all services"
	@echo "  test         - Run tests with race detector"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint         - Run linters"
	@echo "  fmt          - Format code"
	@echo "  run          - Run auth service"
	@echo "  dev          - Run with hot reload"
	@echo "  wire         - Generate Wire DI"
	@echo "  swagger      - Generate Swagger docs"
	@echo "  docker-up    - Start Docker containers"
	@echo "  docker-down  - Stop Docker containers"
```

---

## 9. Testing Strategy

### Unit Test Pattern (Table-Driven)

```go
// internal/auth/usecase/auth_usecase_test.go

package usecase

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestAuthUseCase_Login(t *testing.T) {
    tests := []struct {
        name        string
        email       string
        password    string
        mockSetup   func(*MockUserRepository)
        wantErr     bool
        errContains string
    }{
        {
            name:     "successful login",
            email:    "test@example.com",
            password: "correct-password",
            mockSetup: func(m *MockUserRepository) {
                m.On("GetByEmail", mock.Anything, "test@example.com").
                    Return(&domain.User{
                        Email:        "test@example.com",
                        PasswordHash: "$2a$12$...", // bcrypt hash
                    }, nil)
            },
            wantErr: false,
        },
        {
            name:        "user not found",
            email:       "notfound@example.com",
            password:    "any-password",
            mockSetup:   func(m *MockUserRepository) {
                m.On("GetByEmail", mock.Anything, "notfound@example.com").
                    Return(nil, gorm.ErrRecordNotFound)
            },
            wantErr:     true,
            errContains: "invalid credentials",
        },
        {
            name:     "wrong password",
            email:    "test@example.com",
            password: "wrong-password",
            mockSetup: func(m *MockUserRepository) {
                m.On("GetByEmail", mock.Anything, "test@example.com").
                    Return(&domain.User{
                        Email:        "test@example.com",
                        PasswordHash: "$2a$12$...",
                    }, nil)
            },
            wantErr:     true,
            errContains: "invalid credentials",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockRepo := new(MockUserRepository)
            tt.mockSetup(mockRepo)

            uc := NewAuthUseCase(mockRepo, testConfig)

            // Act
            token, err := uc.Login(context.Background(), tt.email, tt.password)

            // Assert
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errContains)
                assert.Empty(t, token)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, token)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### Integration Test with Test Containers

```go
// test/integration/auth_test.go

//go:build integration

package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type AuthIntegrationSuite struct {
    suite.Suite
    postgresContainer testcontainers.Container
    redisContainer    testcontainers.Container
}

func (s *AuthIntegrationSuite) SetupSuite() {
    ctx := context.Background()

    // Start PostgreSQL container
    pgReq := testcontainers.ContainerRequest{
        Image:        "postgres:15-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
            "POSTGRES_DB":       "test",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections"),
    }
    s.postgresContainer, _ = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: pgReq,
        Started:          true,
    })

    // Start Redis container
    redisReq := testcontainers.ContainerRequest{
        Image:        "redis:7-alpine",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }
    s.redisContainer, _ = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: redisReq,
        Started:          true,
    })
}

func (s *AuthIntegrationSuite) TearDownSuite() {
    s.postgresContainer.Terminate(context.Background())
    s.redisContainer.Terminate(context.Background())
}

func TestAuthIntegration(t *testing.T) {
    suite.Run(t, new(AuthIntegrationSuite))
}
```

---

## 10. Security Checklist

| Item               | Implementation                        | Status |
| ------------------ | ------------------------------------- | ------ |
| SQL Injection      | GORM parameterized queries            | ✅     |
| Password Storage   | bcrypt with cost 12                   | ✅     |
| JWT Security       | HS256 with secret rotation support    | ✅     |
| Rate Limiting      | Redis-backed per-IP limiting          | ✅     |
| CORS               | Configurable origins                  | ✅     |
| Input Validation   | Struct tags + custom validators       | ✅     |
| Error Messages     | No sensitive data in responses        | ✅     |
| Secrets Management | Environment variables                 | ✅     |
| TLS                | Configurable (production requirement) | ⬜     |
| mTLS               | Service-to-service (optional)         | ⬜     |

---

## 11. Learning Resources for Go Newbies

### Key Go Concepts to Learn

1. **Error Handling**: `if err != nil`, error wrapping with `%w`
2. **Interfaces**: Implicit satisfaction, small interfaces
3. **Context**: Cancellation, timeouts, values
4. **Goroutines & Channels**: Concurrent patterns
5. **Struct Tags**: JSON, validation, GORM
6. **Packages**: Import paths, internal packages
7. **Testing**: Table-driven tests, testify

### Recommended Reading

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Proverbs](https://go-proverbs.github.io/)

---

## Summary: What Makes This Production-Grade?

| Feature             | Why It Matters                             |
| ------------------- | ------------------------------------------ |
| Graceful Shutdown   | Zero-downtime deployments in Kubernetes    |
| Health Checks       | Proper orchestration and load balancing    |
| Circuit Breakers    | Prevent cascade failures                   |
| Rate Limiting       | Protect against abuse and DDoS             |
| Structured Logging  | Debug production issues efficiently        |
| Request Tracing     | Correlate requests across services         |
| Prometheus Metrics  | Monitor system health                      |
| Table-Driven Tests  | Comprehensive test coverage                |
| Wire DI             | Catch configuration errors at compile time |
| Context Propagation | Proper timeout and cancellation handling   |

---

**Plan Status:** Ready for implementation
**Estimated Effort:** 6 weeks for full implementation
**Prerequisites:** Go 1.25+, Docker, basic Go knowledge

**Next Step:** Use `superpowers:executing-plans` to begin Phase 1 implementation.
