// Package server provides health check utilities.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthChecker is an interface for health checking.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// HealthCheckerFunc is a function that implements HealthChecker.
type HealthCheckerFunc func(ctx context.Context) error

// Ping implements HealthChecker.
func (f HealthCheckerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	db           HealthChecker
	redis        HealthChecker
	serviceName  string
	version      string
	startTime    time.Time
	customChecks map[string]HealthChecker
}

// HealthHandlerConfig holds health handler configuration.
type HealthHandlerConfig struct {
	ServiceName  string
	Version      string
	DB           HealthChecker
	Redis        HealthChecker
	CustomChecks map[string]HealthChecker
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(cfg HealthHandlerConfig) *HealthHandler {
	return &HealthHandler{
		db:           cfg.DB,
		redis:        cfg.Redis,
		serviceName:  cfg.ServiceName,
		version:      cfg.Version,
		startTime:    time.Now(),
		customChecks: cfg.CustomChecks,
	}
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version,omitempty"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// PublicHealth handles public health check (for load balancers).
// Returns 200 if service is running, no dependency checks.
// @Summary Public health check
// @Description Returns 200 if service is running (for load balancers)
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) PublicHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Service:   h.serviceName,
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// AdminHealth handles admin health check with dependency status.
// @Summary Admin health check
// @Description Detailed health with dependency status
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /admin/health [get]
func (h *HealthHandler) AdminHealth(c *gin.Context) {
	ctx := c.Request.Context()
	status := http.StatusOK
	checks := make(map[string]string)

	// Check database
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = "error: " + err.Error()
			status = http.StatusServiceUnavailable
		} else {
			checks["database"] = "ok"
		}
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			checks["redis"] = "error: " + err.Error()
			status = http.StatusServiceUnavailable
		} else {
			checks["redis"] = "ok"
		}
	}

	// Check custom dependencies
	for name, checker := range h.customChecks {
		if err := checker.Ping(ctx); err != nil {
			checks[name] = "error: " + err.Error()
			status = http.StatusServiceUnavailable
		} else {
			checks[name] = "ok"
		}
	}

	// Determine overall status
	overallStatus := "ok"
	if status != http.StatusOK {
		overallStatus = "degraded"
	}

	c.JSON(status, HealthResponse{
		Status:    overallStatus,
		Service:   h.serviceName,
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).String(),
		Checks:    checks,
	})
}

// ReadyProbe handles Kubernetes readiness probe.
// Returns 200 only if all critical dependencies are healthy.
// @Summary Readiness probe
// @Description Returns 200 if service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 503 {object} map[string]bool
// @Router /ready [get]
func (h *HealthHandler) ReadyProbe(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database (critical)
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ready": false,
				"error": "database unavailable",
			})
			return
		}
	}

	// Check Redis (critical)
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ready": false,
				"error": "redis unavailable",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// LiveProbe handles Kubernetes liveness probe.
// Returns 200 if service is running (doesn't check dependencies).
// @Summary Liveness probe
// @Description Returns 200 if service is alive
// @Tags health
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /live [get]
func (h *HealthHandler) LiveProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

// StartupProbe handles Kubernetes startup probe.
// Returns 200 if service has started successfully.
// @Summary Startup probe
// @Description Returns 200 if service has started
// @Tags health
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /started [get]
func (h *HealthHandler) StartupProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"started": true,
	})
}

// RegisterHealthRoutes registers health check routes on the given router.
func (h *HealthHandler) RegisterHealthRoutes(r *gin.RouterGroup) {
	// Public routes (no auth required)
	r.GET("/health", h.PublicHealth)
	r.GET("/ready", h.ReadyProbe)
	r.GET("/live", h.LiveProbe)
	r.GET("/started", h.StartupProbe)
}

// RegisterAdminHealthRoutes registers admin health routes on the given router.
// These should be protected by admin/system authentication.
func (h *HealthHandler) RegisterAdminHealthRoutes(r *gin.RouterGroup) {
	r.GET("/health", h.AdminHealth)
}

// AddCustomCheck adds a custom health check.
func (h *HealthHandler) AddCustomCheck(name string, checker HealthChecker) {
	if h.customChecks == nil {
		h.customChecks = make(map[string]HealthChecker)
	}
	h.customChecks[name] = checker
}

// RemoveCustomCheck removes a custom health check.
func (h *HealthHandler) RemoveCustomCheck(name string) {
	delete(h.customChecks, name)
}

// Uptime returns the service uptime.
func (h *HealthHandler) Uptime() time.Duration {
	return time.Since(h.startTime)
}

// ServiceName returns the service name.
func (h *HealthHandler) ServiceName() string {
	return h.serviceName
}

// Version returns the service version.
func (h *HealthHandler) Version() string {
	return h.version
}

// RegisterHealthRoutes is a convenience function to register health check routes
// with database and Redis health checkers.
func RegisterHealthRoutes(engine *gin.Engine, db interface{}, redis interface{}) {
	healthHandler := NewHealthHandler(HealthHandlerConfig{
		ServiceName: "service-user",
		Version:     "1.0.0",
		DB:          createDBHealthChecker(db),
		Redis:       createRedisHealthChecker(redis),
	})

	// Register health routes
	engine.GET("/health", healthHandler.PublicHealth)
	engine.GET("/ready", healthHandler.ReadyProbe)
	engine.GET("/live", healthHandler.LiveProbe)
	engine.GET("/started", healthHandler.StartupProbe)
}

// createDBHealthChecker creates a health checker from GORM DB.
func createDBHealthChecker(db interface{}) HealthChecker {
	if db == nil {
		return nil
	}
	return HealthCheckerFunc(func(ctx context.Context) error {
		// Type assertion for *gorm.DB
		if gormDB, ok := db.(*gorm.DB); ok {
			sqlDB, err := gormDB.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		}
		return nil
	})
}

// createRedisHealthChecker creates a health checker from Redis client.
func createRedisHealthChecker(redis interface{}) HealthChecker {
	if redis == nil {
		return nil
	}
	return HealthCheckerFunc(func(ctx context.Context) error {
		// Try to get Redis client's Ping method
		type pinger interface {
			Ping(ctx context.Context) error
		}
		if p, ok := redis.(pinger); ok {
			return p.Ping(ctx)
		}
		return nil
	})
}
