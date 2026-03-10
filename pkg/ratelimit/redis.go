// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"context"
	"fmt"
	"strings"

	"github.com/ignata/go-microservices-boilerplate/pkg/database"
)

// BuildKeyPrefix builds an environment- and service-scoped key prefix.
func BuildKeyPrefix(env, serviceName string) string {
	env = strings.TrimSpace(env)
	if env == "" {
		env = "default"
	}

	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		serviceName = "service"
	}

	return fmt.Sprintf("ratelimit:%s:%s", env, serviceName)
}

// RateLimitResult represents the result of a rate limit check.
type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter int
	Limit      int
}

// RouteLimit defines the rate limit configuration for a specific route.
type RouteLimit struct {
	MaxRequests   int
	WindowSeconds int
}

// RouteRateLimiter provides Redis-backed rate limiting with per-route configuration.
type RouteRateLimiter struct {
	redis     *database.RedisClient
	keyPrefix string
	limits    map[string]RouteLimit
	luaScript string
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter.
func NewRedisRateLimiter(redis *database.RedisClient, keyPrefix string) *RouteRateLimiter {
	luaScript := `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call("INCR", key)
if current == 1 then
  redis.call("EXPIRE", key, window)
end
if current > limit then
  local ttl = redis.call("TTL", key)
  return {0, limit, current, ttl}
else
  return {1, limit - current, current, 0}
end
`
	return &RouteRateLimiter{
		redis:     redis,
		keyPrefix: keyPrefix,
		limits:    make(map[string]RouteLimit),
		luaScript: luaScript,
	}
}

func (r *RouteRateLimiter) SetLimit(route string, maxRequests int, windowSeconds int) {
	r.limits[route] = RouteLimit{
		MaxRequests:   maxRequests,
		WindowSeconds: windowSeconds,
	}
}

func (r *RouteRateLimiter) SetLimits(limits map[string]RouteLimit) {
	for route, limit := range limits {
		r.limits[route] = limit
	}
}

func (r *RouteRateLimiter) GetLimit(route string) (RouteLimit, bool) {
	limit, ok := r.limits[route]
	return limit, ok
}

func (r *RouteRateLimiter) Check(ctx context.Context, key string, limit int, windowSeconds int) (*RateLimitResult, error) {
	fullKey := fmt.Sprintf("%s:%s", r.keyPrefix, key)

	result, err := r.redis.Eval(ctx, r.luaScript, []string{fullKey}, limit, windowSeconds).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	if len(result) != 4 {
		return nil, fmt.Errorf("unexpected result length from rate limit script")
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))
	ttl := int(result[3].(int64))

	retryAfter := 0
	if !allowed {
		if ttl > 0 {
			retryAfter = ttl
		} else {
			retryAfter = windowSeconds
		}
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		Limit:      limit,
	}, nil
}

func (r *RouteRateLimiter) Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	result, err := r.Check(ctx, key, limit, windowSeconds)
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

func (r *RouteRateLimiter) GetKey(clientIP, route string) string {
	return fmt.Sprintf("%s:%s", clientIP, route)
}

func (r *RouteRateLimiter) Ping(ctx context.Context) error {
	return r.redis.Ping(ctx)
}
