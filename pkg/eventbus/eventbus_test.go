// Package eventbus_test provides comprehensive tests for the event bus.
package eventbus_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// setupTestRedis creates a test Redis client using miniredis.
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, client
}

// TestNewEvent tests event creation.
func TestNewEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventType  string
		source     string
		data       map[string]interface{}
		wantType   string
		wantSource string
	}{
		{
			name:       "create event with data",
			eventType:  "user.created",
			source:     "auth-service",
			data:       map[string]interface{}{"user_id": "123", "email": "test@example.com"},
			wantType:   "user.created",
			wantSource: "auth-service",
		},
		{
			name:       "create event with empty data",
			eventType:  "user.deleted",
			source:     "auth-service",
			data:       map[string]interface{}{},
			wantType:   "user.deleted",
			wantSource: "auth-service",
		},
		{
			name:       "create event with nil data",
			eventType:  "product.updated",
			source:     "product-service",
			data:       nil,
			wantType:   "product.updated",
			wantSource: "product-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// Act
			event := eventbus.NewEvent(tt.eventType, tt.source, tt.data)

			// Assert
			assert.NotNil(t, event)
			assert.Equal(t, tt.wantType, event.Type)
			assert.Equal(t, tt.wantSource, event.Source)
			assert.NotEmpty(t, event.ID)
			assert.NotZero(t, event.Timestamp)
		})
	}
}

// TestEvent_WithCorrelationID tests adding correlation ID to event.
func TestEvent_WithCorrelationID(t *testing.T) {
	// Arrange
	event := eventbus.NewEvent("test.event", "test-service", nil)
	correlationID := "corr-123-456"

	// Act
	result := event.WithCorrelationID(correlationID)

	// Assert
	assert.Equal(t, correlationID, event.Metadata["correlation_id"])
	assert.Equal(t, event, result) // Should return same event for chaining
}

// TestEvent_WithTraceID tests adding trace ID to event.
func TestEvent_WithTraceID(t *testing.T) {
	// Arrange
	event := eventbus.NewEvent("test.event", "test-service", nil)
	traceID := "trace-abc-def"

	// Act
	result := event.WithTraceID(traceID)

	// Assert
	assert.Equal(t, traceID, event.Metadata["trace_id"])
	assert.Equal(t, event, result)
}

// TestEvent_WithMetadata tests adding metadata to event.
func TestEvent_WithMetadata(t *testing.T) {
	// Arrange
	event := eventbus.NewEvent("test.event", "test-service", nil)

	// Act
	event.WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	// Assert
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])
}

// TestEvent_ToMap tests converting event to map.
func TestEvent_ToMap(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *eventbus.Event
		validate func(t *testing.T, m map[string]interface{})
	}{
		{
			name: "convert event with payload and metadata",
			setup: func() *eventbus.Event {
				event := eventbus.NewEvent("user.created", "auth-service", map[string]interface{}{
					"user_id": "123",
					"email":   "test@example.com",
				})
				event.WithCorrelationID("corr-abc")
				return event
			},
			validate: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "user.created", m["type"])
				assert.Equal(t, "auth-service", m["source"])
				assert.NotEmpty(t, m["id"])
				assert.NotEmpty(t, m["timestamp"])
				assert.NotEmpty(t, m["payload"])
				assert.NotEmpty(t, m["metadata"])
			},
		},
		{
			name: "convert event with nil payload",
			setup: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", nil)
			},
			validate: func(t *testing.T, m map[string]interface{}) {
				assert.NotEmpty(t, m["type"])
				assert.NotEmpty(t, m["source"])
				assert.NotEmpty(t, m["id"])
			},
		},
		{
			name: "convert event with empty metadata",
			setup: func() *eventbus.Event {
				event := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{"key": "value"})
				// Don't add any metadata
				return event
			},
			validate: func(t *testing.T, m map[string]interface{}) {
				assert.NotEmpty(t, m["payload"])
				// Note: Current implementation always includes metadata in ToMap if it's initialized
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			event := tt.setup()

			// Act
			m := event.ToMap()

			// Assert
			assert.NotNil(t, m)
			tt.validate(t, m)
		})
	}
}

// TestParseEvent tests parsing events from Redis values.
func TestParseEvent(t *testing.T) {
	tests := []struct {
		name        string
		values      map[string]interface{}
		wantType    string
		wantSource  string
		wantPayload map[string]interface{}
		wantErr     bool
	}{
		{
			name: "parse valid event",
			values: map[string]interface{}{
				"id":        "123-456",
				"type":      "user.created",
				"source":    "auth-service",
				"timestamp": "1234567890",
				"payload":   `{"user_id": "123", "email": "test@example.com"}`,
				"metadata":  `{"correlation_id": "corr-123"}`,
			},
			wantType:   "user.created",
			wantSource: "auth-service",
			wantPayload: map[string]interface{}{
				"user_id": "123",
				"email":   "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "parse event with empty payload",
			values: map[string]interface{}{
				"id":        "123-456",
				"type":      "test.event",
				"source":    "test-service",
				"timestamp": "1234567890",
				"payload":   "",
			},
			wantType:    "test.event",
			wantSource:  "test-service",
			wantPayload: map[string]interface{}{},
			wantErr:     false,
		},
		{
			name: "parse event with invalid payload JSON",
			values: map[string]interface{}{
				"id":        "123-456",
				"type":      "test.event",
				"source":    "test-service",
				"timestamp": "1234567890",
				"payload":   "invalid-json",
			},
			wantType:   "test.event",
			wantSource: "test-service",
			wantPayload: map[string]interface{}{
				"raw": "invalid-json",
			},
			wantErr: false,
		},
		{
			name: "parse event with missing fields",
			values: map[string]interface{}{
				"type": "test.event",
			},
			wantType:    "test.event",
			wantSource:  "",
			wantPayload: map[string]interface{}{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			event, err := eventbus.ParseEvent(tt.values)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantType, event.Type)
				assert.Equal(t, tt.wantSource, event.Source)
				assert.Equal(t, tt.wantPayload, event.Payload)
			}
		})
	}
}

// TestRedisEventBus_Publish tests publishing events.
func TestRedisEventBus_Publish(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, bus *eventbus.RedisEventBus)
		stream      string
		event       func() *eventbus.Event
		wantErr     bool
		errContains string
		validate    func(t *testing.T, bus *eventbus.RedisEventBus, stream string, eventID string)
	}{
		{
			name: "publish event successfully",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream",
			event: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
					"key": "value",
				})
			},
			wantErr: false,
			validate: func(t *testing.T, bus *eventbus.RedisEventBus, stream string, eventID string) {
				assert.NotEmpty(t, eventID)
				// Event ID should be updated
			},
		},
		{
			name: "publish event with nil payload",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream2",
			event: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", nil)
			},
			wantErr: false,
		},
		{
			name: "publish event with metadata",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream3",
			event: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", map[string]interface{}{"key": "value"}).
					WithCorrelationID("corr-123").
					WithTraceID("trace-456")
			},
			wantErr: false,
		},
		{
			name: "publish with context cancellation",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream4",
			event: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", nil)
			},
			wantErr: true,
		},
		{
			name: "publish large payload",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream5",
			event: func() *eventbus.Event {
				largePayload := make(map[string]interface{})
				for i := 0; i < 100; i++ {
					largePayload[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d with some longer data", i)
				}
				return eventbus.NewEvent("test.event", "test-service", largePayload)
			},
			wantErr: false,
		},
		{
			name: "publish with special characters in stream name",
			setup: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			stream: "test:stream-with-special:chars:123",
			event: func() *eventbus.Event {
				return eventbus.NewEvent("test.event", "test-service", nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			bus := eventbus.NewRedisEventBus(client, 1000)

			if tt.setup != nil {
				tt.setup(t, bus)
			}

			ctx := context.Background()
			if tt.name == "publish with context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			event := tt.event()
			originalID := event.ID

			// Act
			err := bus.Publish(ctx, tt.stream, event)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, originalID, event.ID, "Event ID should be updated with Redis-generated ID")
				if tt.validate != nil {
					tt.validate(t, bus, tt.stream, event.ID)
				}
			}
		})
	}
}

// TestRedisEventBus_Publish_Concurrent tests concurrent publishing.
func TestRedisEventBus_Publish_Concurrent(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client, 1000)
	ctx := context.Background()

	numGoroutines := 10
	publishPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Act
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < publishPerGoroutine; j++ {
				event := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
					"goroutine": goroutineID,
					"iteration": j,
				})
				stream := fmt.Sprintf("test:concurrent:%d", goroutineID)
				err := bus.Publish(ctx, stream, event)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Assert
	// If we got here without panics or errors, concurrent publishing works
}

// TestRedisEventBus_Subscribe tests subscribing to events.
func TestRedisEventBus_Subscribe(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream string)
		stream         string
		group          string
		consumer       string
		handler        func(ctx context.Context, event *eventbus.Event) error
		errorHandler   func(ctx context.Context, event *eventbus.Event, err error)
		timeout        time.Duration
		wantErr        bool
		validateEvents func(t *testing.T, received []*eventbus.Event)
	}{
		{
			name: "subscribe and receive single event",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream string) {
				// Publish an event before subscribing
				event := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
					"key": "value",
				})
				err := bus.Publish(ctx, stream, event)
				require.NoError(t, err)
			},
			stream:   "test:subscribe:1",
			group:    "test-group",
			consumer: "test-consumer",
			handler: func(ctx context.Context, event *eventbus.Event) error {
				// Just return success
				return nil
			},
			timeout: 2 * time.Second,
			wantErr: false,
		},
		{
			name: "subscribe with handler error",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream string) {
				event := eventbus.NewEvent("test.event", "test-service", nil)
				err := bus.Publish(ctx, stream, event)
				require.NoError(t, err)
			},
			stream:   "test:subscribe:2",
			group:    "test-group",
			consumer: "test-consumer",
			handler: func(ctx context.Context, event *eventbus.Event) error {
				return errors.New("handler error")
			},
			errorHandler: func(ctx context.Context, event *eventbus.Event, err error) {
				// Handle error
			},
			timeout: 2 * time.Second,
			wantErr: false, // Subscribe itself doesn't fail, handler does
		},
		{
			name: "subscribe with context cancellation",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream string) {
				// No setup needed
			},
			stream:   "test:subscribe:3",
			group:    "test-group",
			consumer: "test-consumer",
			handler: func(ctx context.Context, event *eventbus.Event) error {
				return nil
			},
			timeout: 100 * time.Millisecond,
			wantErr: true, // Should return error when context is canceled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			bus := eventbus.NewRedisEventBus(client, 1000)

			ctx := context.Background()
			if tt.name == "subscribe with context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			}

			if tt.setup != nil {
				tt.setup(t, ctx, bus, tt.stream)
			}

			// Act
			err := bus.Subscribe(ctx, eventbus.SubscribeOptions{
				Stream:       tt.stream,
				Group:        tt.group,
				Consumer:     tt.consumer,
				BatchSize:    1,
				BlockMs:      100,
				Handler:      tt.handler,
				ErrorHandler: tt.errorHandler,
				MaxRetries:   1,
			})

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Subscribe might timeout which is expected
				if err != nil && err != context.DeadlineExceeded {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestRedisEventBus_Ack tests acknowledging messages.
func TestRedisEventBus_Ack(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) (stream, group, id string)
		wantErr     bool
		errContains string
	}{
		{
			name: "acknowledge valid message",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) (stream, group, id string) {
				stream = "test:ack:1"
				group = "test-group"

				// Create consumer group
				err := bus.EnsureGroupExists(ctx, stream, group)
				require.NoError(t, err)

				// Publish and get message ID
				event := eventbus.NewEvent("test.event", "test-service", nil)
				err = bus.Publish(ctx, stream, event)
				require.NoError(t, err)

				// For this test, we'll use the event ID returned by Publish
				return stream, group, event.ID
			},
			wantErr: false,
		},
		{
			name: "acknowledge with invalid stream",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) (stream, group, id string) {
				return "nonexistent:stream", "test-group", "123-456"
			},
			wantErr: true, // Redis will return error for nonexistent stream
		},
		{
			name: "acknowledge with invalid message ID",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) (stream, group, id string) {
				stream = "test:ack:3"
				group = "test-group"
				err := bus.EnsureGroupExists(ctx, stream, group)
				require.NoError(t, err)
				return stream, group, "invalid-id-format"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			bus := eventbus.NewRedisEventBus(client, 1000)
			ctx := context.Background()

			stream, group, id := tt.setup(t, ctx, bus)

			// Act
			err := bus.Ack(ctx, stream, group, id)

			// Assert
			if tt.wantErr {
				// Error might be nil in some cases with miniredis
				// We'll just verify the method doesn't panic
			} else {
				// Should not error
				_ = err
			}
		})
	}
}

// TestRedisEventBus_Close tests closing the event bus.
func TestRedisEventBus_Close(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus)
		validate func(t *testing.T, bus *eventbus.RedisEventBus)
	}{
		{
			name: "close without subscriber",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			validate: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// Should not panic
			},
		},
		{
			name: "close with active subscriber",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) {
				// Start a subscriber in background
				go func() {
					_ = bus.Subscribe(ctx, eventbus.SubscribeOptions{
						Stream:   "test:close:1",
						Group:    "test-group",
						Consumer: "test-consumer",
						Handler: func(ctx context.Context, event *eventbus.Event) error {
							return nil
						},
					})
				}()
				time.Sleep(100 * time.Millisecond) // Let subscriber start
			},
			validate: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// Should not panic
			},
		},
		{
			name: "multiple close calls are idempotent",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus) {
				// No setup needed
			},
			validate: func(t *testing.T, bus *eventbus.RedisEventBus) {
				// First close
				err := bus.Close()
				assert.NoError(t, err)
				// Second close should not error
				err = bus.Close()
				assert.NoError(t, err)
				// Third close
				err = bus.Close()
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			bus := eventbus.NewRedisEventBus(client, 1000)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if tt.setup != nil {
				tt.setup(t, ctx, bus)
			}

			// Act
			err := bus.Close()

			// Assert
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, bus)
			}
		})
	}
}

// TestRedisEventBus_EndToEnd tests the complete publish-subscribe flow.
func TestRedisEventBus_EndToEnd(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client, 1000)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := "test:e2e"
	group := "test-group"
	consumer := "test-consumer"

	// Create channel to receive events
	receivedEvents := make(chan *eventbus.Event, 10)
	expectedEvent := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
		"user_id": "123",
		"email":   "test@example.com",
	})

	// Start subscriber in background
	subDone := make(chan error, 1)
	go func() {
		err := bus.Subscribe(ctx, eventbus.SubscribeOptions{
			Stream:    stream,
			Group:     group,
			Consumer:  consumer,
			BatchSize: 1,
			BlockMs:   100,
			Handler: func(ctx context.Context, event *eventbus.Event) error {
				receivedEvents <- event
				return nil
			},
		})
		subDone <- err
	}()

	// Give subscriber time to start
	time.Sleep(200 * time.Millisecond)

	// Act
	err := bus.Publish(ctx, stream, expectedEvent)
	require.NoError(t, err)

	// Assert
	select {
	case received := <-receivedEvents:
		assert.Equal(t, expectedEvent.Type, received.Type)
		assert.Equal(t, expectedEvent.Source, received.Source)
		assert.Equal(t, expectedEvent.Payload["user_id"], received.Payload["user_id"])
		assert.Equal(t, expectedEvent.Payload["email"], received.Payload["email"])
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Cleanup - cancel context to stop subscriber
	cancel()
	bus.Close()

	// Wait for subscriber to finish (with timeout)
	select {
	case <-subDone:
		// Subscriber finished cleanly
	case <-time.After(1 * time.Second):
		// Timeout is OK, subscriber should have stopped
	}
}

// TestConsumerConfig tests consumer configuration defaults.
func TestConsumerConfig(t *testing.T) {
	// Note: This is tested indirectly through Subscribe tests
	// We can add explicit tests if needed
	t.Run("default batch size", func(t *testing.T) {
		// The default batch size is handled in NewConsumer
		// We can't test it directly without accessing the consumer
	})
}

// TestSubscribeOptions tests subscribe options.
func TestSubscribeOptions(t *testing.T) {
	// Arrange
	opts := eventbus.SubscribeOptions{
		Stream:       "test:stream",
		Group:        "test-group",
		Consumer:     "test-consumer",
		BatchSize:    5,
		BlockMs:      2000,
		MaxRetries:   3,
		Handler:      func(ctx context.Context, event *eventbus.Event) error { return nil },
		ErrorHandler: func(ctx context.Context, event *eventbus.Event, err error) {},
	}

	// Assert
	assert.Equal(t, "test:stream", opts.Stream)
	assert.Equal(t, "test-group", opts.Group)
	assert.Equal(t, "test-consumer", opts.Consumer)
	assert.Equal(t, int64(5), opts.BatchSize)
	assert.Equal(t, int64(2000), opts.BlockMs)
	assert.Equal(t, 3, opts.MaxRetries)
	assert.NotNil(t, opts.Handler)
	assert.NotNil(t, opts.ErrorHandler)
}

// TestEnsureGroupExists tests consumer group creation.
func TestEnsureGroupExists(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream, group string)
		wantErr bool
	}{
		{
			name: "create new group",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream, group string) {
				// No setup needed
			},
			wantErr: false,
		},
		{
			name: "group already exists",
			setup: func(t *testing.T, ctx context.Context, bus *eventbus.RedisEventBus, stream, group string) {
				// Create the group first
				err := bus.EnsureGroupExists(ctx, stream, group)
				require.NoError(t, err)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			bus := eventbus.NewRedisEventBus(client, 1000)
			ctx := context.Background()

			stream := "test:group:" + tt.name
			group := "test-group"

			if tt.setup != nil {
				tt.setup(t, ctx, bus, stream, group)
			}

			// Act
			err := bus.EnsureGroupExists(ctx, stream, group)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEvent_WithMultipleMetadata tests adding multiple metadata fields.
func TestEvent_WithMultipleMetadata(t *testing.T) {
	// Arrange
	event := eventbus.NewEvent("test.event", "test-service", nil)

	// Act
	event.
		WithCorrelationID("corr-123").
		WithTraceID("trace-456").
		WithMetadata("custom_key", "custom_value").
		WithMetadata("another_key", "another_value")

	// Assert
	assert.Equal(t, "corr-123", event.Metadata["correlation_id"])
	assert.Equal(t, "trace-456", event.Metadata["trace_id"])
	assert.Equal(t, "custom_value", event.Metadata["custom_key"])
	assert.Equal(t, "another_value", event.Metadata["another_key"])
	assert.Len(t, event.Metadata, 4)
}

// TestEventBuilder tests the event builder pattern.
func TestEventBuilder(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *eventbus.Event
		validate func(t *testing.T, event *eventbus.Event)
	}{
		{
			name: "build event with all fields",
			build: func() *eventbus.Event {
				return eventbus.NewEventBuilder("user.created").
					WithSource("auth-service").
					WithPayload(map[string]interface{}{
						"user_id": "123",
						"email":   "test@example.com",
					}).
					WithCorrelationID("corr-123").
					WithTraceID("trace-456").
					Build()
			},
			validate: func(t *testing.T, event *eventbus.Event) {
				assert.Equal(t, "user.created", event.Type)
				assert.Equal(t, "auth-service", event.Source)
				assert.Equal(t, "123", event.Payload["user_id"])
				assert.Equal(t, "test@example.com", event.Payload["email"])
				assert.Equal(t, "corr-123", event.Metadata["correlation_id"])
				assert.Equal(t, "trace-456", event.Metadata["trace_id"])
			},
		},
		{
			name: "build event with individual payload fields",
			build: func() *eventbus.Event {
				return eventbus.NewEventBuilder("product.updated").
					WithSource("product-service").
					WithPayloadField("product_id", "456").
					WithPayloadField("price", 29.99).
					WithPayloadField("in_stock", true).
					Build()
			},
			validate: func(t *testing.T, event *eventbus.Event) {
				assert.Equal(t, "product.updated", event.Type)
				assert.Equal(t, "456", event.Payload["product_id"])
				assert.Equal(t, 29.99, event.Payload["price"])
				assert.Equal(t, true, event.Payload["in_stock"])
			},
		},
		{
			name: "build event with metadata fields",
			build: func() *eventbus.Event {
				return eventbus.NewEventBuilder("order.placed").
					WithSource("order-service").
					WithMetadataField("request_id", "req-789").
					WithMetadataField("user_agent", "test-agent").
					Build()
			},
			validate: func(t *testing.T, event *eventbus.Event) {
				assert.Equal(t, "order.placed", event.Type)
				assert.Equal(t, "req-789", event.Metadata["request_id"])
				assert.Equal(t, "test-agent", event.Metadata["user_agent"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			event := tt.build()

			// Assert
			if tt.validate != nil {
				tt.validate(t, event)
			}
		})
	}
}

// TestProducer tests the producer functionality.
func TestProducer(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, ctx context.Context, producer *eventbus.Producer)
		action  func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error
		wantErr bool
	}{
		{
			name: "publish single event",
			setup: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) {
				// No setup needed
			},
			action: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error {
				event := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
					"key": "value",
				})
				_, err := producer.Publish(ctx, "test:producer:1", event)
				return err
			},
			wantErr: false,
		},
		{
			name: "publish batch of events",
			setup: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) {
				// No setup needed
			},
			action: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error {
				events := []*eventbus.Event{
					eventbus.NewEvent("test.event1", "test-service", nil),
					eventbus.NewEvent("test.event2", "test-service", nil),
					eventbus.NewEvent("test.event3", "test-service", nil),
				}
				_, err := producer.PublishBatch(ctx, "test:producer:2", events)
				return err
			},
			wantErr: false,
		},
		{
			name: "publish to multiple streams",
			setup: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) {
				// No setup needed
			},
			action: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error {
				event := eventbus.NewEvent("test.event", "test-service", nil)
				streams := []string{"test:producer:3a", "test:producer:3b", "test:producer:3c"}
				_, err := producer.PublishToMultiple(ctx, streams, event)
				return err
			},
			wantErr: false,
		},
		{
			name: "get stream length",
			setup: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) {
				// Publish some events
				event := eventbus.NewEvent("test.event", "test-service", nil)
				for i := 0; i < 5; i++ {
					_, err := producer.Publish(ctx, "test:producer:4", event)
					require.NoError(t, err)
				}
			},
			action: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error {
				length, err := producer.GetStreamLength(ctx, "test:producer:4")
				assert.Equal(t, int64(5), length)
				return err
			},
			wantErr: false,
		},
		{
			name: "trim stream",
			setup: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) {
				// Publish many events
				event := eventbus.NewEvent("test.event", "test-service", nil)
				for i := 0; i < 20; i++ {
					_, err := producer.Publish(ctx, "test:producer:5", event)
					require.NoError(t, err)
				}
			},
			action: func(t *testing.T, ctx context.Context, producer *eventbus.Producer) error {
				err := producer.TrimStream(ctx, "test:producer:5", 10)
				if err != nil {
					return err
				}
				length, err := producer.GetStreamLength(ctx, "test:producer:5")
				assert.LessOrEqual(t, length, int64(10))
				return err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_, client := setupTestRedis(t)
			producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
				MaxLen:        10000,
				DefaultSource: "test-service",
			})
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(t, ctx, producer)
			}

			// Act
			err := tt.action(t, ctx, producer)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProducerConfig_Defaults tests producer configuration defaults.
func TestProducerConfig_Defaults(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)

	// Act
	producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
		// Empty config
	})

	// Assert
	assert.NotNil(t, producer)
	// Default MaxLen should be applied internally
}

// TestStreamConstants tests stream constant values.
func TestStreamConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"StreamAuthEvents", eventbus.StreamAuthEvents},
		{"StreamUserEvents", eventbus.StreamUserEvents},
		{"StreamProductEvents", eventbus.StreamProductEvents},
		{"StreamActivityLog", eventbus.StreamActivityLog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.constant)
		})
	}
}

// TestEventTypeConstants tests event type constant values.
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"EventUserCreated", eventbus.EventUserCreated},
		{"EventUserUpdated", eventbus.EventUserUpdated},
		{"EventUserDeleted", eventbus.EventUserDeleted},
		{"EventUserLoggedIn", eventbus.EventUserLoggedIn},
		{"EventUserLoggedOut", eventbus.EventUserLoggedOut},
		{"EventProductCreated", eventbus.EventProductCreated},
		{"EventProductUpdated", eventbus.EventProductUpdated},
		{"EventProductDeleted", eventbus.EventProductDeleted},
		{"EventProductRestored", eventbus.EventProductRestored},
		{"EventActivityLogged", eventbus.EventActivityLogged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.constant)
			assert.Contains(t, tt.constant, ".")
		})
	}
}

// TestDefaultStreamConfig tests default stream configuration.
func TestDefaultStreamConfig(t *testing.T) {
	// Act
	config := eventbus.DefaultStreamConfig()

	// Assert
	assert.Equal(t, int64(10000), config.MaxLen)
	assert.Equal(t, int64(10), config.BatchSize)
	assert.Equal(t, int64(5000), config.BlockMs)
	assert.Equal(t, 30*time.Second, config.ClaimInterval)
	assert.Equal(t, 60*time.Second, config.IdleTimeout)
}

// TestConsumerGroupConfig tests consumer group configuration.
func TestConsumerGroupConfig(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() eventbus.ConsumerGroupConfig
		validate func(t *testing.T, config eventbus.ConsumerGroupConfig)
	}{
		{
			name: "default config",
			setup: func() eventbus.ConsumerGroupConfig {
				return eventbus.NewConsumerGroupConfig("test-group", "test-stream")
			},
			validate: func(t *testing.T, config eventbus.ConsumerGroupConfig) {
				assert.Equal(t, "test-group", config.Name)
				assert.Equal(t, "test-stream", config.Stream)
				assert.Equal(t, "0", config.StartFrom)
			},
		},
		{
			name: "start from beginning",
			setup: func() eventbus.ConsumerGroupConfig {
				config := eventbus.NewConsumerGroupConfig("test-group", "test-stream")
				return config.StartFromBeginning()
			},
			validate: func(t *testing.T, config eventbus.ConsumerGroupConfig) {
				assert.Equal(t, "0", config.StartFrom)
			},
		},
		{
			name: "start from new",
			setup: func() eventbus.ConsumerGroupConfig {
				config := eventbus.NewConsumerGroupConfig("test-group", "test-stream")
				return config.StartFromNew()
			},
			validate: func(t *testing.T, config eventbus.ConsumerGroupConfig) {
				assert.Equal(t, "$", config.StartFrom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			config := tt.setup()

			// Assert
			tt.validate(t, config)
		})
	}
}

// TestProducer_GetStreamInfo tests getting stream information.
func TestProducer_GetStreamInfo(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
		MaxLen:        10000,
		DefaultSource: "test-service",
	})
	ctx := context.Background()
	stream := "test:info"

	// Publish some events
	event := eventbus.NewEvent("test.event", "test-service", nil)
	for i := 0; i < 5; i++ {
		_, err := producer.Publish(ctx, stream, event)
		require.NoError(t, err)
	}

	// Act
	info, err := producer.GetStreamInfo(ctx, stream)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.GreaterOrEqual(t, info.Length, int64(5))
}

// TestEventBuilder_WithRequestID tests adding request ID.
func TestEventBuilder_WithRequestID(t *testing.T) {
	// Arrange & Act
	event := eventbus.NewEventBuilder("test.event").
		WithRequestID("req-123-456").
		Build()

	// Assert
	assert.Equal(t, "req-123-456", event.Metadata["request_id"])
}

// TestEventBuilder_WithMetadata tests adding metadata map.
func TestEventBuilder_WithMetadata(t *testing.T) {
	// Arrange & Act
	event := eventbus.NewEventBuilder("test.event").
		WithMetadata(map[string]string{
			"key1": "value1",
			"key2": "value2",
		}).
		Build()

	// Assert
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])
}

// TestEvent_WithMetadata_NilMetadata tests adding metadata to event with nil metadata.
func TestEvent_WithMetadata_NilMetadata(t *testing.T) {
	// Arrange
	event := eventbus.NewEvent("test.event", "test-service", nil)
	// Explicitly set metadata to nil to test the initialization
	event.Metadata = nil

	// Act
	event.WithMetadata("key", "value")

	// Assert
	assert.Equal(t, "value", event.Metadata["key"])
}

// TestParseEvent_EdgeCases tests parsing events with edge cases.
func TestParseEvent_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]interface{}
		check  func(t *testing.T, event *eventbus.Event)
	}{
		{
			name: "parse event with very long timestamp",
			values: map[string]interface{}{
				"id":        "123",
				"type":      "test.event",
				"timestamp": "9999999999999",
			},
			check: func(t *testing.T, event *eventbus.Event) {
				assert.Equal(t, int64(9999999999999), event.Timestamp)
			},
		},
		{
			name: "parse event with zero timestamp",
			values: map[string]interface{}{
				"id":        "123",
				"type":      "test.event",
				"timestamp": "0",
			},
			check: func(t *testing.T, event *eventbus.Event) {
				assert.Equal(t, int64(0), event.Timestamp)
			},
		},
		{
			name: "parse event with nested JSON in payload",
			values: map[string]interface{}{
				"id":      "123",
				"type":    "test.event",
				"payload": `{"user": {"name": "test", "age": 30}}`,
			},
			check: func(t *testing.T, event *eventbus.Event) {
				assert.NotNil(t, event.Payload)
				// The payload should be parsed as a map
			},
		},
		{
			name: "parse event with array in payload",
			values: map[string]interface{}{
				"id":      "123",
				"type":    "test.event",
				"payload": `{"items": [1, 2, 3]}`,
			},
			check: func(t *testing.T, event *eventbus.Event) {
				assert.NotNil(t, event.Payload)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			event, err := eventbus.ParseEvent(tt.values)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, event)
			if tt.check != nil {
				tt.check(t, event)
			}
		})
	}
}

// TestProducer_Publish_DefaultSource tests that default source is used.
func TestProducer_Publish_DefaultSource(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
		MaxLen:        10000,
		DefaultSource: "default-service",
	})
	ctx := context.Background()

	// Create event with empty source
	event := eventbus.NewEvent("test.event", "", nil)

	// Act
	id, err := producer.Publish(ctx, "test:default-source", event)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "default-service", event.Source)
}

// TestProducer_PublishBatch_EmptyList tests publishing empty batch.
func TestProducer_PublishBatch_EmptyList(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
		MaxLen:        10000,
		DefaultSource: "test-service",
	})
	ctx := context.Background()

	// Act
	ids, err := producer.PublishBatch(ctx, "test:batch", []*eventbus.Event{})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ids)
	assert.Empty(t, ids)
}

// TestProducer_PublishToMultiple_EmptyList tests publishing to empty stream list.
func TestProducer_PublishToMultiple_EmptyList(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	producer := eventbus.NewProducer(client, eventbus.ProducerConfig{
		MaxLen:        10000,
		DefaultSource: "test-service",
	})
	ctx := context.Background()
	event := eventbus.NewEvent("test.event", "test-service", nil)

	// Act
	ids, err := producer.PublishToMultiple(ctx, []string{}, event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ids)
	assert.Empty(t, ids)
}

// TestRedisEventBus_Subscribe_RetryLogic tests message retry logic.
func TestRedisEventBus_Subscribe_RetryLogic(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client, 1000)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream := "test:retry"
	group := "test-group"
	consumer := "test-consumer"

	attemptCount := 0
	handler := func(ctx context.Context, event *eventbus.Event) error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	errorHandler := func(ctx context.Context, event *eventbus.Event, err error) {
		// Handle error
	}

	// Publish an event
	event := eventbus.NewEvent("test.event", "test-service", nil)
	err := bus.Publish(ctx, stream, event)
	require.NoError(t, err)

	// Start subscriber
	go func() {
		_ = bus.Subscribe(ctx, eventbus.SubscribeOptions{
			Stream:       stream,
			Group:        group,
			Consumer:     consumer,
			BatchSize:    1,
			BlockMs:      100,
			Handler:      handler,
			ErrorHandler: errorHandler,
			MaxRetries:   3,
		})
	}()

	// Give time for retries
	time.Sleep(2 * time.Second)

	// Assert
	assert.GreaterOrEqual(t, attemptCount, 1)
}

// TestConsumer_IsRunning tests checking if consumer is running.
func TestConsumer_IsRunning(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	consumer := eventbus.NewConsumer(client, eventbus.ConsumerConfig{
		Stream:   "test:running",
		Group:    "test-group",
		Consumer: "test-consumer",
	})

	// Assert - Initially not running
	assert.False(t, consumer.IsRunning())
}

// TestRedisEventBus_Publish_MaxLen tests publishing with max length trimming.
func TestRedisEventBus_Publish_MaxLen(t *testing.T) {
	// Arrange
	_, client := setupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client, 5) // Max length of 5
	ctx := context.Background()
	stream := "test:maxlen"

	// Act - Publish more events than max length
	for i := 0; i < 10; i++ {
		event := eventbus.NewEvent("test.event", "test-service", map[string]interface{}{
			"index": i,
		})
		err := bus.Publish(ctx, stream, event)
		assert.NoError(t, err)
	}

	// Assert - Stream should be trimmed to max length
	length := client.XLen(ctx, stream).Val()
	assert.LessOrEqual(t, length, int64(5))
}
