// Package resilience provides resilience patterns for distributed systems.
// It includes circuit breaker and retry implementations.
package resilience

import (
	"time"

	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	// Name is the circuit breaker name (for logging)
	Name string
	// MaxRequests is the maximum number of requests allowed in half-open state
	MaxRequests uint32
	// Timeout is the duration of the open state before transitioning to half-open
	Timeout time.Duration
	// Interval is the period for clearing internal counts
	Interval time.Duration
	// FailureRatio is the ratio of failures that trips the circuit
	FailureRatio float64
	// MinRequests is the minimum number of requests before the circuit can trip
	MinRequests uint32
	// OnStateChange is called when the state changes
	OnStateChange func(name string, from, to gobreaker.State)
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  5,
		Timeout:      30 * time.Second,
		Interval:     60 * time.Second,
		FailureRatio: 0.6,
		MinRequests:  5,
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Timeout:     cfg.Timeout,
		Interval:    cfg.Interval,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= cfg.MinRequests && failureRatio >= cfg.FailureRatio
		},
		OnStateChange: cfg.OnStateChange,
	})
}

// DatabaseCircuitBreakerConfig returns a circuit breaker config optimized for database operations.
func DatabaseCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  5,
		Timeout:      30 * time.Second,
		Interval:     60 * time.Second,
		FailureRatio: 0.6,
		MinRequests:  3,
	}
}

// RedisCircuitBreakerConfig returns a circuit breaker config optimized for Redis operations.
func RedisCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  10,
		Timeout:      15 * time.Second,
		Interval:     30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  5,
	}
}

// HTTPCircuitBreakerConfig returns a circuit breaker config optimized for HTTP clients.
func HTTPCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  3,
		Timeout:      20 * time.Second,
		Interval:     30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState string

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed CircuitBreakerState = "closed"
	// StateOpen means the circuit is open and requests are rejected
	StateOpen CircuitBreakerState = "open"
	// StateHalfOpen means the circuit is allowing test requests
	StateHalfOpen CircuitBreakerState = "half-open"
)

// GetState returns the current state of the circuit breaker.
func GetState(cb *gobreaker.CircuitBreaker) CircuitBreakerState {
	switch cb.State() {
	case gobreaker.StateClosed:
		return StateClosed
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	default:
		return StateClosed
	}
}

// CircuitBreakerGroup manages multiple circuit breakers.
type CircuitBreakerGroup struct {
	breakers map[string]*gobreaker.CircuitBreaker
}

// NewCircuitBreakerGroup creates a new circuit breaker group.
func NewCircuitBreakerGroup() *CircuitBreakerGroup {
	return &CircuitBreakerGroup{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
	}
}

// Add adds a circuit breaker to the group.
func (g *CircuitBreakerGroup) Add(name string, cb *gobreaker.CircuitBreaker) {
	g.breakers[name] = cb
}

// Get retrieves a circuit breaker by name.
func (g *CircuitBreakerGroup) Get(name string) *gobreaker.CircuitBreaker {
	return g.breakers[name]
}

// Execute executes a function through the named circuit breaker.
func (g *CircuitBreakerGroup) Execute(name string, fn func() (interface{}, error)) (interface{}, error) {
	cb := g.breakers[name]
	if cb == nil {
		return fn()
	}
	return cb.Execute(fn)
}

// States returns the state of all circuit breakers.
func (g *CircuitBreakerGroup) States() map[string]CircuitBreakerState {
	states := make(map[string]CircuitBreakerState)
	for name, cb := range g.breakers {
		states[name] = GetState(cb)
	}
	return states
}
