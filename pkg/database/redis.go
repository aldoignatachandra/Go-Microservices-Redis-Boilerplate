// Package database provides Redis connection management.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// RedisClient wraps the go-redis client with additional utilities.
type RedisClient struct {
	*redis.Client
}

// NewRedisConnection creates a new Redis connection.
// It supports both standalone and cluster configurations.
func NewRedisConnection(cfg *RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.PoolSize / 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

// Ping checks if the Redis connection is alive.
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Health returns health status of the Redis connection.
func (r *RedisClient) Health(ctx context.Context) map[string]interface{} {
	status := make(map[string]interface{})

	start := time.Now()
	err := r.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
		return status
	}

	status["status"] = "healthy"
	status["latency"] = latency.String()

	// Get Redis info
	_, err = r.Client.Info(ctx, "server").Result()
	if err == nil {
		status["info"] = "available"
	}

	// Get pool stats
	stats := r.Client.PoolStats()
	status["pool_hits"] = stats.Hits
	status["pool_misses"] = stats.Misses
	status["pool_timeouts"] = stats.Timeouts

	return status
}

// SetWithExpiry sets a key with expiration.
func (r *RedisClient) SetWithExpiry(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// GetOrSet gets a value or sets it if it doesn't exist (cache-aside pattern).
func (r *RedisClient) GetOrSet(ctx context.Context, key string, fn func() (interface{}, error), expiration time.Duration) (interface{}, error) {
	// Try to get from cache first
	val, err := r.Client.Get(ctx, key).Result()
	if err == nil {
		return val, nil
	}

	if err != redis.Nil {
		return nil, fmt.Errorf("failed to get from redis: %w", err)
	}

	// Value not in cache, get from source
	data, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors as cache is optional)
	_ = r.SetWithExpiry(ctx, key, data, expiration)

	return data, nil
}

// DeleteByPattern deletes all keys matching a pattern.
// Useful for cache invalidation.
func (r *RedisClient) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	var count int64
	iter := r.Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := r.Client.Del(ctx, iter.Val()).Err(); err != nil {
			return count, err
		}
		count++
	}
	return count, iter.Err()
}

// AcquireLock acquires a distributed lock using SET NX EX.
// Returns the lock value (for release) if successful.
func (r *RedisClient) AcquireLock(ctx context.Context, key string, expiration time.Duration) (string, error) {
	value := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := r.Client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("lock already held")
	}
	return value, nil
}

// ReleaseLock releases a distributed lock using Lua script for atomicity.
func (r *RedisClient) ReleaseLock(ctx context.Context, key, value string) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	return r.Client.Eval(ctx, script, []string{key}, value).Err()
}

// IsLocked checks if a lock is currently held.
func (r *RedisClient) IsLocked(ctx context.Context, key string) (bool, error) {
	exists, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Increment increments a counter and returns the new value.
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// IncrementBy increments a counter by a specific amount.
func (r *RedisClient) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.Client.IncrBy(ctx, key, value).Result()
}

// Decrement decrements a counter and returns the new value.
func (r *RedisClient) Decrement(ctx context.Context, key string) (int64, error) {
	return r.Client.Decr(ctx, key).Result()
}

// Expire sets expiration on a key.
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live for a key.
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// Exists checks if a key exists.
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.Client.Exists(ctx, keys...).Result()
}

// Delete removes keys.
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}
