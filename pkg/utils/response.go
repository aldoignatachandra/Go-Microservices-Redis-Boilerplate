// Package utils provides common utilities for HTTP responses.
package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// AdminAccessRequiredMessage is the canonical message for admin-only endpoint access denial.
	AdminAccessRequiredMessage = "forbidden: admin access required"
)

// Response is the standard API response format (aligned with Bun-Hono).
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// ErrorBody represents an error in the response.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains metadata about the response.
type Meta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

// ListMeta contains metadata for list responses.
type ListMeta struct {
	Count      int         `json:"count"`
	Pagination *Pagination `json:"pagination"`
	Meta       *Meta       `json:"meta"`
}

// Pagination contains pagination information (camelCase for Bun-Hono alignment).
type Pagination struct {
	Page            int   `json:"page"`
	Limit           int   `json:"limit"`
	Total           int64 `json:"total"`
	TotalPages      int   `json:"totalPages"`
	HasNextPage     bool  `json:"hasNextPage"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}

// Success sends a successful response with optional message.
func Success(c *gin.Context, statusCode int, data interface{}, message ...string) {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	c.JSON(statusCode, Response{
		Success: true,
		Message: msg,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: getRequestID(c),
		},
	})
}

// Created sends a 201 Created response.
func Created(c *gin.Context, data interface{}, message ...string) {
	Success(c, http.StatusCreated, data, message...)
}

// OK sends a 200 OK response.
func OK(c *gin.Context, data interface{}, message ...string) {
	Success(c, http.StatusOK, data, message...)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// List sends a list response with pagination.
func List(c *gin.Context, data interface{}, _, limit int, total int64) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: getRequestID(c),
		},
	})
}

// Error sends an error response.
func Error(c *gin.Context, statusCode int, code, message string, details ...string) {
	errBody := &ErrorBody{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		errBody.Details = details[0]
	}

	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Data:    errBody,
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: getRequestID(c),
		},
	})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details...)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, resource string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}

// ValidationError sends a 422 Unprocessable Entity response.
func ValidationError(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, details...)
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// ServiceUnavailable sends a 503 Service Unavailable response.
func ServiceUnavailable(c *gin.Context, message string) {
	Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message)
}

// Gone sends a 410 Gone response (for soft-deleted resources).
func Gone(c *gin.Context, resource string) {
	Error(c, http.StatusGone, "GONE", resource+" has been deleted")
}

// TooManyRequests sends a 429 Too Many Requests response.
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "rate limit exceeded"
	}
	Error(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", message)
}

// ErrorResponse sends an error response with optional error details.
// This is a convenience function that extracts error details if provided.
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	details := ""
	if err != nil {
		details = err.Error()
	}
	Error(c, statusCode, "ERROR", message, details)
}

// getRequestID extracts the request ID from context.
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		if str, ok := id.(string); ok {
			return str
		}
	}
	return ""
}

// Paginate calculates pagination values.
func Paginate(page, limit int) (offset int, normalizedPage int, normalizedLimit int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return (page - 1) * limit, page, limit
}

// CalculatePagination creates a Pagination struct.
func CalculatePagination(page, limit int, total int64) *Pagination {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &Pagination{
		Page:            page,
		Limit:           limit,
		Total:           total,
		TotalPages:      totalPages,
		HasNextPage:     page < totalPages,
		HasPreviousPage: page > 1,
	}
}
