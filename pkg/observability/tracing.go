// Package observability provides distributed tracing and observability utilities.
package observability

import (
	"context"

	"github.com/google/uuid"
)

const (
	// TraceIDKey is the context key for trace ID.
	TraceIDKey = "trace_id"
	// SpanIDKey is the context key for span ID.
	SpanIDKey = "span_id"
	// ParentSpanIDKey is the context key for parent span ID.
	ParentSpanIDKey = "parent_span_id"
)

// Span represents a trace span.
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Operation    string
	StartTime    int64
	EndTime      int64
	Tags         map[string]string
}

// SpanOption is a function that configures a span.
type SpanOption func(*Span)

// WithTag adds a tag to the span.
func WithTag(key, value string) SpanOption {
	return func(s *Span) {
		if s.Tags == nil {
			s.Tags = make(map[string]string)
		}
		s.Tags[key] = value
	}
}

// Tracer manages trace spans.
type Tracer struct {
	serviceName string
}

// NewTracer creates a new tracer.
func NewTracer(serviceName string) *Tracer {
	return &Tracer{
		serviceName: serviceName,
	}
}

// StartSpan starts a new span.
func (t *Tracer) StartSpan(ctx context.Context, operation string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		TraceID:   getTraceID(ctx),
		SpanID:    uuid.New().String(),
		Operation: operation,
		Tags:      make(map[string]string),
	}

	// Get parent span ID if exists
	if parentSpanID, ok := ctx.Value(SpanIDKey).(string); ok {
		span.ParentSpanID = parentSpanID
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Add service name tag
	span.Tags["service"] = t.serviceName

	// Set span in context
	ctx = context.WithValue(ctx, TraceIDKey, span.TraceID)
	ctx = context.WithValue(ctx, SpanIDKey, span.SpanID)

	return ctx, span
}

// Finish completes a span.
func (t *Tracer) Finish(span *Span) {
	// In a real implementation, this would export the span
	// to a tracing backend like Jaeger, Zipkin, or OpenTelemetry
	_ = span
}

// getTraceID gets or creates a trace ID from context.
func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}
	return uuid.New().String()
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID retrieves the span ID from context.
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}
