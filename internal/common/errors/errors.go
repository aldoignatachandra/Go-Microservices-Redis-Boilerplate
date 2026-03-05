// Package errors provides shared error types and utilities across all microservices.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents a structured application error with HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Common application errors.
var (
	ErrNotFound       = &AppError{Code: http.StatusNotFound, Message: "resource not found"}
	ErrBadRequest     = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrUnauthorized   = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden      = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrConflict       = &AppError{Code: http.StatusConflict, Message: "resource already exists"}
	ErrInternalServer = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrValidation     = &AppError{Code: http.StatusUnprocessableEntity, Message: "validation error"}
)

// NewNotFoundError creates a not found error with a custom message.
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// NewBadRequestError creates a bad request error with a custom message.
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

// NewValidationError creates a validation error with a custom message.
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
	}
}

// NewConflictError creates a conflict error with a custom message.
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:    http.StatusConflict,
		Message: message,
	}
}

// NewInternalError wraps an internal error.
func NewInternalError(err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
		Err:     err,
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from an error chain.
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == http.StatusNotFound
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == http.StatusUnauthorized
}

// IsConflict checks if an error is a conflict error.
func IsConflict(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == http.StatusConflict
}
