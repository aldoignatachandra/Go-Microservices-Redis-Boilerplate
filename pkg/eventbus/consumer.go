// Package eventbus provides consumer implementation for Redis Streams.
package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ConsumerConfig holds consumer configuration.
type ConsumerConfig struct {
	// Stream is the Redis stream name
	Stream string
	// Group is the consumer group name
	Group string
	// Consumer is the consumer name within the group
	Consumer string
	// BatchSize is the number of messages to fetch per read
	BatchSize int64
	// BlockMs is the blocking time in milliseconds for XREADGROUP
	BlockMs int64
	// MaxRetries is the maximum number of retry attempts (0 = no retry)
	MaxRetries int
}

// Consumer consumes events from a Redis stream using consumer groups.
type Consumer struct {
	client    *redis.Client
	config    ConsumerConfig
	stopChan  chan struct{}
	running   bool
	mu        sync.Mutex
}

// NewConsumer creates a new stream consumer.
func NewConsumer(client *redis.Client, config ConsumerConfig) *Consumer {
	// Set defaults
	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}
	if config.BlockMs <= 0 {
		config.BlockMs = 5000
	}

	return &Consumer{
		client:   client,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Consume starts consuming messages from the stream.
// It blocks until the context is cancelled or Stop() is called.
func (c *Consumer) Consume(
	ctx context.Context,
	handler func(ctx context.Context, event *Event) error,
	errorHandler func(ctx context.Context, event *Event, err error),
) error {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	// Ensure consumer group exists
	if err := c.ensureGroupExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure consumer group: %w", err)
	}

	// Start claiming pending messages in background
	go c.claimPendingMessages(ctx, handler, errorHandler)

	// Main consumption loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		default:
			if err := c.readAndProcess(ctx, handler, errorHandler); err != nil {
				if errorHandler != nil {
					errorHandler(ctx, nil, err)
				}
				// Brief pause before retrying
				time.Sleep(time.Second)
			}
		}
	}
}

// readAndProcess reads a batch of messages and processes them.
func (c *Consumer) readAndProcess(
	ctx context.Context,
	handler func(ctx context.Context, event *Event) error,
	errorHandler func(ctx context.Context, event *Event, err error),
) error {
	// Read messages from stream
	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.config.Group,
		Consumer: c.config.Consumer,
		Streams:  []string{c.config.Stream, ">"},
		Count:    c.config.BatchSize,
		Block:    time.Duration(c.config.BlockMs) * time.Millisecond,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			// No new messages, this is normal
			return nil
		}
		return fmt.Errorf("failed to read from stream: %w", err)
	}

	// Process each message
	for _, stream := range streams {
		for _, message := range stream.Messages {
			c.processMessage(ctx, message, handler, errorHandler)
		}
	}

	return nil
}

// processMessage processes a single message with retry logic.
func (c *Consumer) processMessage(
	ctx context.Context,
	message redis.XMessage,
	handler func(ctx context.Context, event *Event) error,
	errorHandler func(ctx context.Context, event *Event, err error),
) {
	event, err := ParseEvent(message.Values)
	if err != nil {
		if errorHandler != nil {
			errorHandler(ctx, nil, fmt.Errorf("failed to parse event: %w", err))
		}
		// Ack the malformed message to avoid reprocessing
		_ = c.client.XAck(ctx, c.config.Stream, c.config.Group, message.ID)
		return
	}

	// Process with retry
	var lastErr error
	maxAttempts := c.config.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := handler(ctx, event); err != nil {
			lastErr = err
			if attempt < maxAttempts {
				// Exponential backoff
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		} else {
			// Success - acknowledge the message
			_ = c.client.XAck(ctx, c.config.Stream, c.config.Group, message.ID)
			return
		}
	}

	// All retries failed
	if errorHandler != nil {
		errorHandler(ctx, event, fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr))
	}

	// Ack to prevent infinite retry (or move to dead letter queue in production)
	_ = c.client.XAck(ctx, c.config.Stream, c.config.Group, message.ID)
}

// claimPendingMessages claims and processes pending messages.
// This handles the case where a consumer crashed before acknowledging.
func (c *Consumer) claimPendingMessages(
	ctx context.Context,
	handler func(ctx context.Context, event *Event) error,
	errorHandler func(ctx context.Context, event *Event, err error),
) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.claimAndProcess(ctx, handler, errorHandler)
		}
	}
}

// claimAndProcess claims pending messages that are older than idle timeout.
func (c *Consumer) claimAndProcess(
	ctx context.Context,
	handler func(ctx context.Context, event *Event) error,
	errorHandler func(ctx context.Context, event *Event, err error),
) {
	// Get pending messages
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.config.Stream,
		Group:  c.config.Group,
		Start:  "-",
		End:    "+",
		Count:  c.config.BatchSize,
	}).Result()

	if err != nil && err != redis.Nil {
		return
	}

	// Claim messages idle for more than 60 seconds
	for _, p := range pending {
		if p.Idle < 60*time.Second {
			continue
		}

		// Claim the message
		claimed, err := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   c.config.Stream,
			Group:    c.config.Group,
			Consumer: c.config.Consumer,
			MinIdle:  60 * time.Second,
			Messages: []string{p.ID},
		}).Result()

		if err != nil {
			continue
		}

		// Process claimed messages
		for _, message := range claimed {
			c.processMessage(ctx, message, handler, errorHandler)
		}
	}
}

// ensureGroupExists creates the consumer group if it doesn't exist.
func (c *Consumer) ensureGroupExists(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.config.Stream, c.config.Group, "0").Err()
	if err != nil && !isBusyGroupError(err) {
		return err
	}
	return nil
}

// Stop stops the consumer.
func (c *Consumer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		c.running = false
		close(c.stopChan)
	}
}

// IsRunning returns whether the consumer is running.
func (c *Consumer) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}
