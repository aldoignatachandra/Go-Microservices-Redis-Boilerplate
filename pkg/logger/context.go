// Package logger provides context-aware logging utilities.
// It extracts contextual information (request ID, user ID, trace ID)
// from context and includes them in log entries.
package logger

import (
	"context"

	"go.uber.org/zap"
)

// ContextKey defines the type for context keys.
type ContextKey string

// Predefined context keys for common values.
const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user ID.
	UserIDKey ContextKey = "user_id"
	// TraceIDKey is the context key for distributed trace ID.
	TraceIDKey ContextKey = "trace_id"
	// ServiceNameKey is the context key for service name.
	ServiceNameKey ContextKey = "service_name"
	// MethodKey is the context key for HTTP method.
	MethodKey ContextKey = "method"
	// PathKey is the context key for request path.
	PathKey ContextKey = "path"
	// StatusCodeKey is the context key for HTTP status code.
	StatusCodeKey ContextKey = "status_code"
)

// ContextValues holds all contextual values for logging.
type ContextValues struct {
	RequestID   string
	UserID      string
	TraceID     string
	ServiceName string
	Method      string
	Path        string
	StatusCode  int
}

// WithContext creates a logger enriched with values from context.
// This enables request tracing and user context in logs.
//
// Example:
//
//	log := logger.WithContext(ctx, logger.L())
//	log.Info("user action", zap.String("action", "login"))
func WithContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if base == nil {
		base = L()
	}

	fields := make([]zap.Field, 0, 7)

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			fields = append(fields, zap.String("request_id", id))
		}
	}

	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			fields = append(fields, zap.String("user_id", id))
		}
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok && id != "" {
			fields = append(fields, zap.String("trace_id", id))
		}
	}

	if serviceName := ctx.Value(ServiceNameKey); serviceName != nil {
		if name, ok := serviceName.(string); ok && name != "" {
			fields = append(fields, zap.String("service", name))
		}
	}

	if method := ctx.Value(MethodKey); method != nil {
		if m, ok := method.(string); ok && m != "" {
			fields = append(fields, zap.String("method", m))
		}
	}

	if path := ctx.Value(PathKey); path != nil {
		if p, ok := path.(string); ok && p != "" {
			fields = append(fields, zap.String("path", p))
		}
	}

	if statusCode := ctx.Value(StatusCodeKey); statusCode != nil {
		if code, ok := statusCode.(int); ok {
			fields = append(fields, zap.Int("status_code", code))
		}
	}

	if len(fields) == 0 {
		return base
	}

	return base.With(fields...)
}

// WithContextValues creates a logger with explicit context values.
// Use this when you have ContextValues struct already populated.
func WithContextValues(base *zap.Logger, cv *ContextValues) *zap.Logger {
	if base == nil {
		base = L()
	}

	if cv == nil {
		return base
	}

	fields := make([]zap.Field, 0, 7)

	if cv.RequestID != "" {
		fields = append(fields, zap.String("request_id", cv.RequestID))
	}
	if cv.UserID != "" {
		fields = append(fields, zap.String("user_id", cv.UserID))
	}
	if cv.TraceID != "" {
		fields = append(fields, zap.String("trace_id", cv.TraceID))
	}
	if cv.ServiceName != "" {
		fields = append(fields, zap.String("service", cv.ServiceName))
	}
	if cv.Method != "" {
		fields = append(fields, zap.String("method", cv.Method))
	}
	if cv.Path != "" {
		fields = append(fields, zap.String("path", cv.Path))
	}
	if cv.StatusCode > 0 {
		fields = append(fields, zap.Int("status_code", cv.StatusCode))
	}

	if len(fields) == 0 {
		return base
	}

	return base.With(fields...)
}

// GetContextValues extracts all context values into a struct.
func GetContextValues(ctx context.Context) *ContextValues {
	cv := &ContextValues{}

	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			cv.RequestID = s
		}
	}

	if v := ctx.Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok {
			cv.UserID = s
		}
	}

	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			cv.TraceID = s
		}
	}

	if v := ctx.Value(ServiceNameKey); v != nil {
		if s, ok := v.(string); ok {
			cv.ServiceName = s
		}
	}

	if v := ctx.Value(MethodKey); v != nil {
		if s, ok := v.(string); ok {
			cv.Method = s
		}
	}

	if v := ctx.Value(PathKey); v != nil {
		if s, ok := v.(string); ok {
			cv.Path = s
		}
	}

	if v := ctx.Value(StatusCodeKey); v != nil {
		if i, ok := v.(int); ok {
			cv.StatusCode = i
		}
	}

	return cv
}

// SetRequestID sets the request ID in context.
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// SetUserID sets the user ID in context.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// SetTraceID sets the trace ID in context.
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// SetServiceName sets the service name in context.
func SetServiceName(ctx context.Context, serviceName string) context.Context {
	return context.WithValue(ctx, ServiceNameKey, serviceName)
}

// SetMethod sets the HTTP method in context.
func SetMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, MethodKey, method)
}

// SetPath sets the request path in context.
func SetPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, PathKey, path)
}

// SetStatusCode sets the status code in context.
func SetStatusCode(ctx context.Context, statusCode int) context.Context {
	return context.WithValue(ctx, StatusCodeKey, statusCode)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
