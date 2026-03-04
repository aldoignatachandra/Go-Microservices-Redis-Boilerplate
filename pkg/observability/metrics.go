// Package observability provides metrics and monitoring utilities.
package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsLabels holds common metric labels.
type MetricsLabels struct {
	Service   string
	Endpoint  string
	Method    string
	Status    string
	ErrorCode string
}

// BusinessMetrics tracks business-level metrics.
type BusinessMetrics struct {
	// User registrations
	Registrations *prometheus.CounterVec
	// User logins
	Logins *prometheus.CounterVec
	// Active users
	ActiveUsers *prometheus.GaugeVec
	// Orders created
	OrdersCreated *prometheus.CounterVec
	// Revenue
	Revenue *prometheus.CounterVec
}

// NewBusinessMetrics creates business metrics.
func NewBusinessMetrics(serviceName string) *BusinessMetrics {
	return &BusinessMetrics{
		Registrations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "user_registrations_total",
				Help:        "Total number of user registrations",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"method"},
		),
		Logins: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "user_logins_total",
				Help:        "Total number of user logins",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"status"},
		),
		ActiveUsers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "active_users_total",
				Help:        "Number of active users",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"window"},
		),
		OrdersCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "orders_created_total",
				Help:        "Total number of orders created",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"status"},
		),
		Revenue: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "revenue_total",
				Help:        "Total revenue in cents",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"currency", "payment_method"},
		),
	}
}

// Register registers all business metrics with the default registry.
func (m *BusinessMetrics) Register() error {
	collectors := []prometheus.Collector{
		m.Registrations,
		m.Logins,
		m.ActiveUsers,
		m.OrdersCreated,
		m.Revenue,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	return nil
}

// InstrumentationMetrics tracks service instrumentation metrics.
type InstrumentationMetrics struct {
	// Database connection pool
	DBConnections *prometheus.GaugeVec
	// Cache hit rate
	CacheHits *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	// Queue depth
	QueueDepth *prometheus.GaugeVec
	// External API calls
	ExternalAPICalls *prometheus.CounterVec
	// External API latency
	ExternalAPILatency *prometheus.HistogramVec
}

// NewInstrumentationMetrics creates instrumentation metrics.
func NewInstrumentationMetrics(serviceName string) *InstrumentationMetrics {
	return &InstrumentationMetrics{
		DBConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "db_connections_total",
				Help:        "Number of database connections",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"state"}, // idle, in_use, open
		),
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "cache_hits_total",
				Help:        "Total number of cache hits",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"cache"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "cache_misses_total",
				Help:        "Total number of cache misses",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"cache"},
		),
		QueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "queue_depth",
				Help:        "Current depth of processing queue",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"queue_name"},
		),
		ExternalAPICalls: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "external_api_calls_total",
				Help:        "Total number of external API calls",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"service", "status"},
		),
		ExternalAPILatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "external_api_latency_seconds",
				Help:        "External API call latency in seconds",
				ConstLabels: prometheus.Labels{"service": serviceName},
				Buckets:     []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service"},
		),
	}
}

// Register registers all instrumentation metrics.
func (m *InstrumentationMetrics) Register() error {
	collectors := []prometheus.Collector{
		m.DBConnections,
		m.CacheHits,
		m.CacheMisses,
		m.QueueDepth,
		m.ExternalAPICalls,
		m.ExternalAPILatency,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	return nil
}

// TimeExternalAPICall times an external API call and records metrics.
func (m *InstrumentationMetrics) TimeExternalAPICall(serviceName string, status string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		m.ExternalAPICalls.WithLabelValues(serviceName, status).Inc()
		m.ExternalAPILatency.WithLabelValues(serviceName).Observe(duration)
	}
}

// HealthMetrics tracks health-related metrics.
type HealthMetrics struct {
	// Health check status
	HealthStatus *prometheus.GaugeVec
	// Dependency health
	DependencyHealth *prometheus.GaugeVec
}

// NewHealthMetrics creates health metrics.
func NewHealthMetrics(serviceName string) *HealthMetrics {
	return &HealthMetrics{
		HealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "health_status",
				Help:        "Health status of the service (1=healthy, 0=unhealthy)",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"check_type"},
		),
		DependencyHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "dependency_health",
				Help:        "Health status of dependencies (1=healthy, 0=unhealthy)",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"dependency"},
		),
	}
}

// Register registers all health metrics.
func (m *HealthMetrics) Register() error {
	collectors := []prometheus.Collector{
		m.HealthStatus,
		m.DependencyHealth,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	return nil
}

// SetHealthStatus sets the health status for a check type.
func (m *HealthMetrics) SetHealthStatus(checkType string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.HealthStatus.WithLabelValues(checkType).Set(value)
}

// SetDependencyHealth sets the health status for a dependency.
func (m *HealthMetrics) SetDependencyHealth(dependency string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.DependencyHealth.WithLabelValues(dependency).Set(value)
}
