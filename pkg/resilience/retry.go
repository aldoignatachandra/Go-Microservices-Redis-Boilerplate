// Package resilience provides retry utilities with exponential backoff.
package resilience

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the initial delay before first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier is the factor by which delay increases
	Multiplier float64
	// Jitter adds randomness to prevent thundering herd
	Jitter bool
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func() error

// RetryableFuncWithResult is a function that returns a result and can be retried.
type RetryableFuncWithResult[T any] func() (T, error)

// IsRetryable determines if an error should trigger a retry.
type IsRetryable func(err error) bool

// Retry executes a function with retry logic.
func Retry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	return RetryWithChecker(ctx, cfg, fn, AlwaysRetry)
}

// RetryWithChecker executes a function with retry logic and custom error checking.
func RetryWithChecker(ctx context.Context, cfg RetryConfig, fn RetryableFunc, isRetryable IsRetryable) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return err
		}

		// Don't sleep after last attempt
		if attempt < cfg.MaxAttempts {
			// Calculate delay with jitter
			sleepDelay := delay
			if cfg.Jitter {
				sleepDelay = addJitter(delay)
			}

			// Wait or context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleepDelay):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// RetryWithResult executes a function with retry logic and returns a result.
func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn RetryableFuncWithResult[T]) (T, error) {
	return RetryWithResultAndChecker(ctx, cfg, fn, AlwaysRetry)
}

// RetryWithResultAndChecker executes a function with retry logic, custom error checking, and returns a result.
func RetryWithResultAndChecker[T any](ctx context.Context, cfg RetryConfig, fn RetryableFuncWithResult[T], isRetryable IsRetryable) (T, error) {
	var result T
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		// Execute function
		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return result, err
		}

		// Don't sleep after last attempt
		if attempt < cfg.MaxAttempts {
			// Calculate delay with jitter
			sleepDelay := delay
			if cfg.Jitter {
				sleepDelay = addJitter(delay)
			}

			// Wait or context cancellation
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(sleepDelay):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return result, fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// AlwaysRetry always returns true (retry all errors).
func AlwaysRetry(err error) bool {
	return true
}

// NeverRetry never retries (retry no errors).
func NeverRetry(err error) bool {
	return false
}

// addJitter adds random jitter to the delay.
func addJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return delay
	}
	// Add between 0% and 50% jitter
	jitter := time.Duration(rand.Float64() * 0.5 * float64(delay))
	return delay + jitter
}

// RetryableError wraps an error to indicate it's retryable.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	var retryable *RetryableError
	return err != nil && (err == retryable || err.Error() != "")
}

// PermanentError wraps an error to indicate it should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewPermanentError creates a new permanent error.
func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

// IsPermanentError checks if an error is permanent.
func IsPermanentError(err error) bool {
	var permanent *PermanentError
	return err != nil && (err == permanent || err.Error() != "")
}

// Backoff provides exponential backoff with optional jitter.
type Backoff struct {
	initial    time.Duration
	max        time.Duration
	multiplier float64
	jitter     bool
	attempt    int
}

// NewBackoff creates a new backoff instance.
func NewBackoff(initial, max time.Duration, multiplier float64, jitter bool) *Backoff {
	return &Backoff{
		initial:    initial,
		max:        max,
		multiplier: multiplier,
		jitter:     jitter,
		attempt:    0,
	}
}

// Next returns the next backoff duration.
func (b *Backoff) Next() time.Duration {
	b.attempt++
	shiftAmount := uint(b.attempt - 1)
	delay := time.Duration(float64(b.initial) * float64(int64(1)<<shiftAmount))
	if b.multiplier > 0 {
		delay = time.Duration(float64(delay) * b.multiplier)
	}
	if delay > b.max {
		delay = b.max
	}
	if b.jitter {
		delay = addJitter(delay)
	}
	return delay
}

// Reset resets the backoff to initial state.
func (b *Backoff) Reset() {
	b.attempt = 0
}

// Attempt returns the current attempt number.
func (b *Backoff) Attempt() int {
	return b.attempt
}
