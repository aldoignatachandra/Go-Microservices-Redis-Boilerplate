// Package metrics provides Prometheus metrics for microservices.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric names following Prometheus naming conventions.
const (
	Namespace = "microservice"
	Subsystem = "http"
)

var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path"},
	)

	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"service", "method", "path"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"service", "method", "path"},
	)

	httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "requests_in_flight",
			Help:      "Current number of HTTP requests being processed",
		},
		[]string{"service", "method"},
	)

	// Redis metrics
	redisPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "redis",
			Name:      "publish_total",
			Help:      "Total events published to Redis",
		},
		[]string{"service", "stream"},
	)

	redisConsumeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "redis",
			Name:      "consume_total",
			Help:      "Total events consumed from Redis",
		},
		[]string{"service", "stream", "status"},
	)

	redisConsumeErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "redis",
			Name:      "consume_errors_total",
			Help:      "Total errors consuming events from Redis",
		},
		[]string{"service", "stream", "error_type"},
	)

	// Database metrics
	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "db",
			Name:      "query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"service", "operation"},
	)

	dbConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "db",
			Name:      "connections",
			Help:      "Current database connections",
		},
		[]string{"service", "state"},
	)

	// Business metrics
	businessOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "business",
			Name:      "operations_total",
			Help:      "Total business operations",
		},
		[]string{"service", "operation", "status"},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpRequestSize)
	prometheus.MustRegister(httpResponseSize)
	prometheus.MustRegister(httpRequestsInFlight)
	prometheus.MustRegister(redisPublishTotal)
	prometheus.MustRegister(redisConsumeTotal)
	prometheus.MustRegister(redisConsumeErrors)
	prometheus.MustRegister(dbQueryDuration)
	prometheus.MustRegister(dbConnections)
	prometheus.MustRegister(businessOperationsTotal)
}

// PrometheusHandler returns the Prometheus HTTP handler.
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// MetricsMiddleware records HTTP metrics for all requests.
func MetricsMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Track in-flight requests
		httpRequestsInFlight.WithLabelValues(serviceName, method).Inc()
		defer httpRequestsInFlight.WithLabelValues(serviceName, method).Dec()

		// Track request size
		if c.Request.ContentLength > 0 {
			httpRequestSize.WithLabelValues(serviceName, method, path).
				Observe(float64(c.Request.ContentLength))
		}

		// Track duration
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		httpRequestsTotal.WithLabelValues(serviceName, method, path, status).Inc()
		httpRequestDuration.WithLabelValues(serviceName, method, path).Observe(duration)
		httpResponseSize.WithLabelValues(serviceName, method, path).Observe(float64(c.Writer.Size()))
	}
}

// RecordRedisPublish records a Redis publish event.
func RecordRedisPublish(serviceName, stream string) {
	redisPublishTotal.WithLabelValues(serviceName, stream).Inc()
}

// RecordRedisConsume records a Redis consume event.
func RecordRedisConsume(serviceName, stream, status string) {
	redisConsumeTotal.WithLabelValues(serviceName, stream, status).Inc()
}

// RecordRedisConsumeError records a Redis consume error.
func RecordRedisConsumeError(serviceName, stream, errorType string) {
	redisConsumeErrors.WithLabelValues(serviceName, stream, errorType).Inc()
}

// RecordDBQuery records a database query duration.
func RecordDBQuery(serviceName, operation string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(serviceName, operation).Observe(duration.Seconds())
}

// SetDBConnections sets the current database connection count.
func SetDBConnections(serviceName, state string, count float64) {
	dbConnections.WithLabelValues(serviceName, state).Set(count)
}

// RecordBusinessOperation records a business operation.
func RecordBusinessOperation(serviceName, operation, status string) {
	businessOperationsTotal.WithLabelValues(serviceName, operation, status).Inc()
}

// IncHTTPRequestsTotal increments the HTTP requests counter.
func IncHTTPRequestsTotal(serviceName, method, path, status string) {
	httpRequestsTotal.WithLabelValues(serviceName, method, path, status).Inc()
}

// ObserveHTTPRequestDuration observes HTTP request duration.
func ObserveHTTPRequestDuration(serviceName, method, path string, duration time.Duration) {
	httpRequestDuration.WithLabelValues(serviceName, method, path).Observe(duration.Seconds())
}

// GetRegistry returns the default Prometheus registry.
func GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}
