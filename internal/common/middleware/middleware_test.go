// Package middleware provides tests for common middleware utilities.
package middleware_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/internal/common/middleware"
)

func TestPaginationParams_GetPage(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected int
	}{
		{"default when zero", 0, 1},
		{"default when negative", -1, 1},
		{"valid page 1", 1, 1},
		{"valid page 5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Page: tt.page}
			assert.Equal(t, tt.expected, p.GetPage())
		})
	}
}

func TestPaginationParams_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{"default when zero", 0, 20},
		{"default when negative", -1, 20},
		{"valid limit 10", 10, 10},
		{"valid limit 50", 50, 50},
		{"capped at max 100", 200, 100},
		{"exact max", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Limit: tt.limit}
			assert.Equal(t, tt.expected, p.GetLimit())
		})
	}
}

func TestPaginationParams_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		limit    int
		expected int
	}{
		{"page 1 limit 20", 1, 20, 0},
		{"page 2 limit 20", 2, 20, 20},
		{"page 3 limit 10", 3, 10, 20},
		{"page 1 limit 0 (default 20)", 1, 0, 0},
		{"page 0 (default 1) limit 20", 0, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &middleware.PaginationParams{Page: tt.page, Limit: tt.limit}
			assert.Equal(t, tt.expected, p.GetOffset())
		})
	}
}
