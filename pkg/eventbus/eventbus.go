// Package eventbus provides an event-driven communication layer using Redis Streams.
// It implements pub/sub patterns with consumer groups for reliable message delivery.
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Event represents a domain event that can be published/consumed.
type Event struct {
	// ID is the unique identifier of the event
	ID string `json:"id"`
	// Type is the event type (e.g., "user.created", "order.completed")
	Type string `json:"type"`
	// Source is the service that published the event
	Source string `json:"source"`
	// Timestamp is when the event occurred
	Timestamp int64 `json:"timestamp"`
	// Payload contains the event data
	Payload map[string]interface{} `json:"payload"`
	// Metadata contains optional additional information
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with generated ID and current timestamp.
func NewEvent(eventType, source string, payload map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
		Metadata:  make(map[string]string),
	}
}

// WithMetadata adds metadata to the event.
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithCorrelationID adds a correlation ID to the event metadata.
func (e *Event) WithCorrelationID(correlationID string) *Event {
	return e.WithMetadata("correlation_id", correlationID)
}

// WithTraceID adds a trace ID to the event metadata.
func (e *Event) WithTraceID(traceID string) *Event {
	return e.WithMetadata("trace_id", traceID)
}

// ToMap converts the event to a map for Redis storage.
func (e *Event) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":        e.ID,
		"type":      e.Type,
		"source":    e.Source,
		"timestamp": fmt.Sprintf("%d", e.Timestamp),
	}

	// Marshal payload to JSON string
	if e.Payload != nil {
		payloadBytes, _ := json.Marshal(e.Payload)
		result["payload"] = string(payloadBytes)
	}

	// Marshal metadata to JSON string
	if len(e.Metadata) > 0 {
		metadataBytes, _ := json.Marshal(e.Metadata)
		result["metadata"] = string(metadataBytes)
	}

	return result
}

// ParseEvent parses an event from Redis stream message values.
func ParseEvent(values map[string]interface{}) (*Event, error) {
	event := &Event{
		Payload:  make(map[string]interface{}),
		Metadata: make(map[string]string),
	}

	if id, ok := values["id"].(string); ok {
		event.ID = id
	}

	if eventType, ok := values["type"].(string); ok {
		event.Type = eventType
	}

	if source, ok := values["source"].(string); ok {
		event.Source = source
	}

	if timestamp, ok := values["timestamp"].(string); ok {
		var ts int64
		if _, err := fmt.Sscanf(timestamp, "%d", &ts); err == nil {
			event.Timestamp = ts
		}
	}

	if payload, ok := values["payload"].(string); ok && payload != "" {
		if err := json.Unmarshal([]byte(payload), &event.Payload); err != nil {
			// Try to handle as raw string
			event.Payload = map[string]interface{}{"raw": payload}
		}
	}

	if metadata, ok := values["metadata"].(string); ok && metadata != "" {
		if err := json.Unmarshal([]byte(metadata), &event.Metadata); err != nil {
			event.Metadata = make(map[string]string)
		}
	}

	return event, nil
}

// EventBus defines the interface for pub/sub operations.
type EventBus interface {
	// Publish sends an event to a Redis stream.
	Publish(ctx context.Context, stream string, event *Event) error

	// Subscribe consumes events from a stream using consumer groups.
	Subscribe(ctx context.Context, opts SubscribeOptions) error

	// Ack acknowledges successful processing of a message.
	Ack(ctx context.Context, stream, group, id string) error

	// Close gracefully shuts down the event bus.
	Close() error
}

// SubscribeOptions configures the consumer behavior.
type SubscribeOptions struct {
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
	// Handler is called for each received event
	Handler func(ctx context.Context, event *Event) error
	// ErrorHandler is called when an error occurs
	ErrorHandler func(ctx context.Context, event *Event, err error)
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
}

// RedisEventBus implements EventBus using Redis Streams.
type RedisEventBus struct {
	client   *redis.Client
	maxLen   int64
	consumer *Consumer
}

// NewRedisEventBus creates a new Redis-based event bus.
func NewRedisEventBus(client *redis.Client, maxLen int64) *RedisEventBus {
	return &RedisEventBus{
		client: client,
		maxLen: maxLen,
	}
}

// Publish sends an event to a Redis stream using XADD.
func (b *RedisEventBus) Publish(ctx context.Context, stream string, event *Event) error {
	args := &redis.XAddArgs{
		Stream: stream,
		Values: event.ToMap(),
	}

	// Limit stream length to prevent unbounded growth
	if b.maxLen > 0 {
		args.MaxLen = b.maxLen
		args.Approx = true // Use ~ for efficiency
	}

	id, err := b.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish event to stream %s: %w", stream, err)
	}

	// Update event ID with Redis-generated ID
	event.ID = id
	return nil
}

// Subscribe starts consuming events from a stream using consumer groups.
func (b *RedisEventBus) Subscribe(ctx context.Context, opts SubscribeOptions) error {
	consumer := NewConsumer(b.client, ConsumerConfig{
		Stream:     opts.Stream,
		Group:      opts.Group,
		Consumer:   opts.Consumer,
		BatchSize:  opts.BatchSize,
		BlockMs:    opts.BlockMs,
		MaxRetries: opts.MaxRetries,
	})

	b.consumer = consumer
	return consumer.Consume(ctx, opts.Handler, opts.ErrorHandler)
}

// Ack acknowledges successful processing of a message.
func (b *RedisEventBus) Ack(ctx context.Context, stream, group, id string) error {
	return b.client.XAck(ctx, stream, group, id).Err()
}

// Close gracefully shuts down the event bus.
func (b *RedisEventBus) Close() error {
	if b.consumer != nil {
		b.consumer.Stop()
	}
	return nil
}

// EnsureGroupExists creates a consumer group if it doesn't exist.
func (b *RedisEventBus) EnsureGroupExists(ctx context.Context, stream, group string) error {
	// Try to create the group, ignore "BUSYGROUP" error
	err := b.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !isBusyGroupError(err) {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// isBusyGroupError checks if the error is a BUSYGROUP error.
func isBusyGroupError(err error) bool {
	return err != nil && (err.Error() == "BUSYGROUP Consumer Group name already exists" ||
		err.Error() == "BUSYGROUP")
}
