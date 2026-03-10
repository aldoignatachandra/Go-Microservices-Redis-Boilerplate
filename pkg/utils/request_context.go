package utils

import "context"

type requestContextKey string

const (
	requestContextKeyIPAddress     requestContextKey = "ip_address"
	requestContextKeyUserAgent     requestContextKey = "user_agent"
	requestContextKeyRequestID     requestContextKey = "request_id"
	requestContextKeyCorrelationID requestContextKey = "correlation_id"
	requestContextKeyActorUserID   requestContextKey = "actor_user_id"
)

// WithRequestContextMetadata stores common request metadata in context.
func WithRequestContextMetadata(ctx context.Context, ipAddress, userAgent, requestID, correlationID string) context.Context {
	ctx = context.WithValue(ctx, requestContextKeyIPAddress, ipAddress)
	ctx = context.WithValue(ctx, requestContextKeyUserAgent, userAgent)
	ctx = context.WithValue(ctx, requestContextKeyRequestID, requestID)
	ctx = context.WithValue(ctx, requestContextKeyCorrelationID, correlationID)
	return ctx
}

// GetIPAddressFromContext returns the request client IP from context.
func GetIPAddressFromContext(ctx context.Context) string {
	return getRequestContextString(ctx, requestContextKeyIPAddress)
}

// GetUserAgentFromContext returns the request user-agent from context.
func GetUserAgentFromContext(ctx context.Context) string {
	return getRequestContextString(ctx, requestContextKeyUserAgent)
}

// GetRequestIDFromContext returns the request ID from context.
func GetRequestIDFromContext(ctx context.Context) string {
	return getRequestContextString(ctx, requestContextKeyRequestID)
}

// GetCorrelationIDFromContext returns the correlation ID from context.
func GetCorrelationIDFromContext(ctx context.Context) string {
	return getRequestContextString(ctx, requestContextKeyCorrelationID)
}

// WithActorUserID stores the actor user ID in context.
func WithActorUserID(ctx context.Context, actorUserID string) context.Context {
	return context.WithValue(ctx, requestContextKeyActorUserID, actorUserID)
}

// GetActorUserIDFromContext returns the actor user ID from context.
func GetActorUserIDFromContext(ctx context.Context) string {
	return getRequestContextString(ctx, requestContextKeyActorUserID)
}

func getRequestContextString(ctx context.Context, key requestContextKey) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return value
}
