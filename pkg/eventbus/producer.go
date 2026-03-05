// Package eventbus provides producer implementation for Redis Streams.
package eventbus

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// EventPublisher defines the interface for publishing events.
// This allows mocking the producer in tests.
type EventPublisher interface {
	Publish(ctx context.Context, stream string, event *Event) (string, error)
}

// Producer publishes events to Redis streams.
type Producer struct {
	client *redis.Client
	config ProducerConfig
}

// ProducerConfig holds producer configuration.
type ProducerConfig struct {
	// MaxLen is the maximum length of the stream (uses approximate trimming)
	MaxLen int64
	// DefaultSource is the default source for events
	DefaultSource string
}

// NewProducer creates a new event producer.
func NewProducer(client *redis.Client, config ProducerConfig) *Producer {
	if config.MaxLen <= 0 {
		config.MaxLen = 10000
	}
	return &Producer{
		client: client,
		config: config,
	}
}

// Publish publishes an event to a stream.
func (p *Producer) Publish(ctx context.Context, stream string, event *Event) (string, error) {
	// Set default source if not specified
	if event.Source == "" {
		event.Source = p.config.DefaultSource
	}

	args := &redis.XAddArgs{
		Stream: stream,
		Values: event.ToMap(),
		MaxLen: p.config.MaxLen,
		Approx: true, // Use ~ for efficiency
	}

	id, err := p.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to publish event: %w", err)
	}

	return id, nil
}

// PublishBatch publishes multiple events to a stream.
func (p *Producer) PublishBatch(ctx context.Context, stream string, events []*Event) ([]string, error) {
	ids := make([]string, len(events))

	// Use pipeline for efficiency
	pipe := p.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(events))

	for i, event := range events {
		if event.Source == "" {
			event.Source = p.config.DefaultSource
		}

		cmds[i] = pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: event.ToMap(),
			MaxLen: p.config.MaxLen,
			Approx: true,
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to publish batch: %w", err)
	}

	for i, cmd := range cmds {
		ids[i] = cmd.Val()
	}

	return ids, nil
}

// PublishToMultiple publishes an event to multiple streams.
func (p *Producer) PublishToMultiple(ctx context.Context, streams []string, event *Event) (map[string]string, error) {
	if event.Source == "" {
		event.Source = p.config.DefaultSource
	}

	ids := make(map[string]string)

	// Use pipeline for efficiency
	pipe := p.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, stream := range streams {
		cmds[stream] = pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: event.ToMap(),
			MaxLen: p.config.MaxLen,
			Approx: true,
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to publish to multiple streams: %w", err)
	}

	for stream, cmd := range cmds {
		ids[stream] = cmd.Val()
	}

	return ids, nil
}

// GetStreamLength returns the length of a stream.
func (p *Producer) GetStreamLength(ctx context.Context, stream string) (int64, error) {
	return p.client.XLen(ctx, stream).Result()
}

// TrimStream trims a stream to a maximum length.
func (p *Producer) TrimStream(ctx context.Context, stream string, maxLen int64) error {
	return p.client.XTrimMaxLen(ctx, stream, maxLen).Err()
}

// GetStreamInfo returns information about a stream.
func (p *Producer) GetStreamInfo(ctx context.Context, stream string) (*redis.XInfoStream, error) {
	return p.client.XInfoStream(ctx, stream).Result()
}
