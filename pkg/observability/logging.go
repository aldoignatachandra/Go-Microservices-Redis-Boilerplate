// Package observability provides structured logging with context.
package observability

import (
	"context"

	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"go.uber.org/zap"
)

// LogContext provides structured logging with context.
type LogContext struct {
	logger logger.Logger
}

// NewLogContext creates a new logging context.
func NewLogContext(log logger.Logger) *LogContext {
	return &LogContext{
		logger: log,
	}
}

// WithContext creates logger fields from context.
func (lc *LogContext) WithContext(ctx context.Context) []zap.Field {
	fields := []zap.Field{}

	// Add trace ID if available
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// Add span ID if available
	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	// Add request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// Add user ID from context if available
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	return fields
}

// Info logs an info message with context fields.
func (lc *LogContext) Info(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(lc.WithContext(ctx), fields...)
	lc.logger.Info(msg, allFields...)
}

// Error logs an error message with context fields.
func (lc *LogContext) Error(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(lc.WithContext(ctx), fields...)
	lc.logger.Error(msg, allFields...)
}

// Warn logs a warning message with context fields.
func (lc *LogContext) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(lc.WithContext(ctx), fields...)
	lc.logger.Warn(msg, allFields...)
}

// Debug logs a debug message with context fields.
func (lc *LogContext) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(lc.WithContext(ctx), fields...)
	lc.logger.Debug(msg, allFields...)
}

// WithFields returns a logger with additional fields.
func (lc *LogContext) WithFields(fields ...zap.Field) *LogContext {
	return &LogContext{
		logger: lc.logger.With(fields...),
	}
}

// HTTPRequest logs an HTTP request with context.
func (lc *LogContext) HTTPRequest(ctx context.Context, method, path, query, clientIP, userAgent string) {
	fields := append(lc.WithContext(ctx),
		zap.String("method", method),
		zap.String("path", path),
		zap.String("query", query),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)
	lc.logger.Info("HTTP request", fields...)
}

// HTTPResponse logs an HTTP response with context.
func (lc *LogContext) HTTPResponse(ctx context.Context, method, path string, status int, latencyMs int64) {
	fields := append(lc.WithContext(ctx),
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", status),
		zap.Int64("latency_ms", latencyMs),
	)
	lc.logger.Info("HTTP response", fields...)
}

// DatabaseQuery logs a database query with context.
func (lc *LogContext) DatabaseQuery(ctx context.Context, table, operation string, durationMs int64) {
	fields := append(lc.WithContext(ctx),
		zap.String("table", table),
		zap.String("operation", operation),
		zap.Int64("duration_ms", durationMs),
	)
	lc.logger.Debug("Database query", fields...)
}

// ExternalAPICall logs an external API call with context.
func (lc *LogContext) ExternalAPICall(ctx context.Context, service, endpoint, method string, status int, durationMs int64) {
	fields := append(lc.WithContext(ctx),
		zap.String("service", service),
		zap.String("endpoint", endpoint),
		zap.String("method", method),
		zap.Int("status", status),
		zap.Int64("duration_ms", durationMs),
	)
	lc.logger.Info("External API call", fields...)
}

// CacheOperation logs a cache operation with context.
func (lc *LogContext) CacheOperation(ctx context.Context, operation, key string, hit bool) {
	fields := append(lc.WithContext(ctx),
		zap.String("operation", operation),
		zap.String("key", key),
		zap.Bool("hit", hit),
	)
	lc.logger.Debug("Cache operation", fields...)
}

// ErrorWithContext logs an error with context.
func (lc *LogContext) ErrorWithContext(ctx context.Context, err error, msg string) {
	fields := append(lc.WithContext(ctx), zap.Error(err))
	lc.logger.Error(msg, fields...)
}
