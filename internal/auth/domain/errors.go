// Package domain provides domain errors for the auth service.
package domain

import "errors"

// Common domain errors.
var (
	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserDeleted        = errors.New("user has been deleted")
	ErrUserInactive       = errors.New("user is inactive")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrEmailAlreadyUsed   = errors.New("email already in use")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrInvalidToken    = errors.New("invalid token")

	// Validation errors
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidRole      = errors.New("invalid role")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// Authorization errors
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrAdminRequired = errors.New("admin privileges required")
)

// Error codes for API responses.
const (
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrSessionNotFound)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrInvalidCredentials) ||
		errors.Is(err, ErrInvalidToken) ||
		errors.Is(err, ErrSessionExpired) ||
		errors.Is(err, ErrSessionRevoked)
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidEmail) ||
		errors.Is(err, ErrInvalidRole) ||
		errors.Is(err, ErrPasswordTooShort)
}
