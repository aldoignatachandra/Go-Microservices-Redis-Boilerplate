// Package constants provides shared constants across all microservices.
package constants

// Service names for event bus and inter-service communication.
const (
	AuthService    = "auth-service"
	UserService    = "user-service"
	ProductService = "product-service"
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
	ContextKeyUserID        contextKey = "user_id"
	ContextKeyUserRole      contextKey = "user_role"
	ContextKeyRequestID     contextKey = "request_id"
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
