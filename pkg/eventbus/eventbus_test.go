// Package eventbus provides tests for the event bus producer.
package eventbus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

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
			event := eventbus.NewEvent(tt.eventType, tt.source, tt.data)

			assert.NotNil(t, event)
			assert.Equal(t, tt.wantType, event.Type)
			assert.Equal(t, tt.wantSource, event.Source)
			assert.NotEmpty(t, event.ID)
			assert.NotZero(t, event.Timestamp) // Timestamp is int64 (unix millis), not time.Time
		})
	}
}

func TestEvent_WithCorrelationID(t *testing.T) {
	event := eventbus.NewEvent("test.event", "test-service", nil)

	correlationID := "corr-123-456"
	result := event.WithCorrelationID(correlationID)

	// CorrelationID is stored in Metadata, not as a top-level field
	assert.Equal(t, correlationID, event.Metadata["correlation_id"])
	assert.Equal(t, event, result) // Should return same event for chaining
}

func TestEvent_ToMap(t *testing.T) {
	event := eventbus.NewEvent("user.created", "auth-service", map[string]interface{}{
		"user_id": "123",
		"email":   "test@example.com",
	})
	event.WithCorrelationID("corr-abc")

	m := event.ToMap()

	assert.NotNil(t, m)
	assert.Equal(t, "user.created", m["type"])
	assert.Equal(t, "auth-service", m["source"])
	assert.NotEmpty(t, m["id"])
	assert.NotEmpty(t, m["timestamp"])

	// correlation_id is inside the metadata JSON blob, not a top-level key
	assert.NotEmpty(t, m["metadata"])
}

func TestStreamConstants(t *testing.T) {
	// Verify stream names are consistent
	assert.NotEmpty(t, eventbus.StreamAuthEvents)
	assert.NotEmpty(t, eventbus.StreamUserEvents)
	assert.NotEmpty(t, eventbus.StreamProductEvents)
}

func TestProducerConfig(t *testing.T) {
	config := eventbus.ProducerConfig{
		MaxLen:        10000,
		DefaultSource: "test-service",
	}

	assert.Equal(t, int64(10000), config.MaxLen)
	assert.Equal(t, "test-service", config.DefaultSource)
}

func TestEvent_Immutability(t *testing.T) {
	// Test that event ID and timestamp are set at creation time
	event1 := eventbus.NewEvent("test", "service", nil)
	time.Sleep(1 * time.Millisecond)
	event2 := eventbus.NewEvent("test", "service", nil)

	assert.NotEqual(t, event1.ID, event2.ID)
	// Both should have timestamps set (int64 unix millis, not time.Time)
	assert.NotZero(t, event1.Timestamp)
	assert.NotZero(t, event2.Timestamp)
}
