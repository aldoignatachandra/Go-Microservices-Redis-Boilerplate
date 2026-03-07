// Package domain provides domain errors for the user service.
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
	ErrEmailAlreadyUsed   = errors.New("email already in use")

	// Profile errors
	ErrProfileNotFound = errors.New("profile not found")

	// Activity errors
	ErrActivityNotFound = errors.New("activity not found")

	// Validation errors
	ErrValidationError  = errors.New("validation error")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidRole      = errors.New("invalid role")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// Authorization errors
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrAdminRequired = errors.New("admin privileges required")
)

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrProfileNotFound) ||
		errors.Is(err, ErrActivityNotFound)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrInvalidCredentials) ||
		errors.Is(err, ErrUnauthorized)
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidEmail) ||
		errors.Is(err, ErrInvalidRole) ||
		errors.Is(err, ErrPasswordTooShort)
}
