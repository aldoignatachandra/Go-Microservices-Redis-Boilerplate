// Package constants provides shared constants across all microservices.
package constants

// Service names for event bus and inter-service communication.
const (
	AuthService    = "service-auth"
	UserService    = "service-user"
	ProductService = "service-product"
)

// HTTP header keys.
const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
)

// Context keys for passing values through request context.
type contextKey string

const (
	// ContextKeyUserID is the context key for user ID.
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyUserRole is the context key for user role.
	ContextKeyUserRole contextKey = "user_role"
	// ContextKeyRequestID is the context key for request ID.
	ContextKeyRequestID contextKey = "request_id"
	// ContextKeyCorrelationID is the context key for correlation ID.
	ContextKeyCorrelationID contextKey = "correlation_id"
)

// Pagination defaults.
const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// User roles.
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// User statuses.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusDeleted  = "deleted"
)
