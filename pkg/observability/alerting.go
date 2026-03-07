// Package observability provides alerting and monitoring utilities.
package observability

import (
	"context"
	"sync"
	"time"

	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"go.uber.org/zap"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	// SeverityInfo represents an informational alert.
	SeverityInfo AlertSeverity = "info"
	// SeverityWarning represents a warning alert.
	SeverityWarning AlertSeverity = "warning"
	// SeverityError represents an error alert.
	SeverityError AlertSeverity = "error"
	// SeverityCritical represents a critical alert.
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents an alert.
type Alert struct {
	ID        string
	Title     string
	Message   string
	Severity  AlertSeverity
	Timestamp time.Time
	Labels    map[string]string
	Metadata  map[string]interface{}
}

// AlertHandler handles alerts.
type AlertHandler interface {
	Handle(ctx context.Context, alert *Alert) error
}

// LogAlertHandler logs alerts to the configured logger.
type LogAlertHandler struct {
	logger logger.Logger
}

// NewLogAlertHandler creates a new log alert handler.
func NewLogAlertHandler(log logger.Logger) *LogAlertHandler {
	return &LogAlertHandler{
		logger: log,
	}
}

// Handle logs the alert.
func (h *LogAlertHandler) Handle(ctx context.Context, alert *Alert) error {
	fields := []zap.Field{
		zap.String("alert_id", alert.ID),
		zap.String("severity", string(alert.Severity)),
		zap.Time("timestamp", alert.Timestamp),
	}

	// Add labels
	for k, v := range alert.Labels {
		fields = append(fields, zap.String(k, v))
	}

	switch alert.Severity {
	case SeverityCritical:
		h.logger.Error(alert.Title, append(fields, zap.String("message", alert.Message))...)
	case SeverityError:
		h.logger.Error(alert.Title, append(fields, zap.String("message", alert.Message))...)
	case SeverityWarning:
		h.logger.Warn(alert.Title, append(fields, zap.String("message", alert.Message))...)
	default:
		h.logger.Info(alert.Title, append(fields, zap.String("message", alert.Message))...)
	}

	return nil
}

// AlertRule defines when an alert should be triggered.
type AlertRule struct {
	Name        string
	Description string
	Severity    AlertSeverity
	Evaluator   func() (*Alert, bool)
	Interval    time.Duration
}

// AlertManager manages alert rules and handlers.
type AlertManager struct {
	rules    []*AlertRule
	handlers []AlertHandler
	logger   logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(log logger.Logger) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertManager{
		rules:    make([]*AlertRule, 0),
		handlers: make([]AlertHandler, 0),
		logger:   log,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// AddRule adds an alert rule.
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// AddHandler adds an alert handler.
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// Start starts the alert manager.
func (am *AlertManager) Start() {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, rule := range am.rules {
		am.wg.Add(1)
		go am.evaluateRule(rule)
	}
}

// Stop stops the alert manager.
func (am *AlertManager) Stop() {
	am.cancel()
	am.wg.Wait()
}

// evaluateRule continuously evaluates a rule.
func (am *AlertManager) evaluateRule(rule *AlertRule) {
	defer am.wg.Done()

	ticker := time.NewTicker(rule.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			if alert, triggered := rule.Evaluator(); triggered {
				am.sendAlert(alert)
			}
		}
	}
}

// sendAlert sends an alert to all handlers.
func (am *AlertManager) sendAlert(alert *Alert) {
	am.mu.RLock()
	handlers := make([]AlertHandler, len(am.handlers))
	copy(handlers, am.handlers)
	am.mu.RUnlock()

	for _, handler := range handlers {
		go func(h AlertHandler) {
			if err := h.Handle(am.ctx, alert); err != nil {
				am.logger.Error("Failed to handle alert",
					zap.String("alert_id", alert.ID),
					zap.Error(err),
				)
			}
		}(handler)
	}
}

// Common alert rules.

// HighErrorRateRule checks if the error rate is too high.
type HighErrorRateRule struct {
	errorCount int64
	totalCount int64
	threshold  float64
	window     time.Duration
	lastReset  time.Time
	mu         sync.Mutex
}

// NewHighErrorRateRule creates a new high error rate rule.
func NewHighErrorRateRule(threshold float64, window time.Duration) *HighErrorRateRule {
	return &HighErrorRateRule{
		threshold: threshold,
		window:    window,
		lastReset: time.Now(),
	}
}

// Record records a request.
func (r *HighErrorRateRule) Record(error bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset if window has passed
	if time.Since(r.lastReset) > r.window {
		r.errorCount = 0
		r.totalCount = 0
		r.lastReset = time.Now()
	}

	r.totalCount++
	if error {
		r.errorCount++
	}
}

// GetRate returns the current error rate.
func (r *HighErrorRateRule) GetRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.totalCount == 0 {
		return 0
	}
	return float64(r.errorCount) / float64(r.totalCount)
}

// ServiceHealthRule checks service health.
type ServiceHealthRule struct {
	isHealthy func() bool
}

// NewServiceHealthRule creates a new service health rule.
func NewServiceHealthRule(isHealthy func() bool) *ServiceHealthRule {
	return &ServiceHealthRule{
		isHealthy: isHealthy,
	}
}
