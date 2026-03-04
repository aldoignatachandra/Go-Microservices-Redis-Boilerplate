// Package eventbus provides configuration options.
package eventbus

import "time"

// StreamConfig holds stream configuration.
type StreamConfig struct {
	// Name is the stream name
	Name string
	// MaxLen is the maximum stream length
	MaxLen int64
	// ConsumerGroup is the consumer group name
	ConsumerGroup string
	// ConsumerName is the consumer name
	ConsumerName string
	// BatchSize is the number of messages to read per batch
	BatchSize int64
	// BlockMs is the blocking time in milliseconds
	BlockMs int64
	// ClaimInterval is the interval for claiming pending messages
	ClaimInterval time.Duration
	// IdleTimeout is the idle time before a message can be claimed
	IdleTimeout time.Duration
}

// DefaultStreamConfig returns default stream configuration.
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		MaxLen:        10000,
		BatchSize:     10,
		BlockMs:       5000,
		ClaimInterval: 30 * time.Second,
		IdleTimeout:   60 * time.Second,
	}
}

// ConsumerGroupConfig holds consumer group configuration.
type ConsumerGroupConfig struct {
	// Name is the consumer group name
	Name string
	// Stream is the stream name
	Stream string
	// StartFrom is the starting position ("0" for beginning, "$" for new messages only)
	StartFrom string
}

// NewConsumerGroupConfig creates a new consumer group configuration.
func NewConsumerGroupConfig(name, stream string) ConsumerGroupConfig {
	return ConsumerGroupConfig{
		Name:      name,
		Stream:    stream,
		StartFrom: "0", // Start from beginning by default
	}
}

// StartFromBeginning sets the consumer to start from the beginning.
func (c ConsumerGroupConfig) StartFromBeginning() ConsumerGroupConfig {
	c.StartFrom = "0"
	return c
}

// StartFromNew sets the consumer to only receive new messages.
func (c ConsumerGroupConfig) StartFromNew() ConsumerGroupConfig {
	c.StartFrom = "$"
	return c
}

// EventTypes defines common event type constants.
const (
	// User events
	EventUserCreated   = "user.created"
	EventUserUpdated   = "user.updated"
	EventUserDeleted   = "user.deleted"
	EventUserLoggedIn  = "user.logged_in"
	EventUserLoggedOut = "user.logged_out"

	// Product events
	EventProductCreated  = "product.created"
	EventProductUpdated  = "product.updated"
	EventProductDeleted  = "product.deleted"
	EventProductRestored = "product.restored"

	// Activity events
	EventActivityLogged = "activity.logged"
)

// StreamNames defines common stream name constants.
const (
	StreamAuthEvents  = "auth:events"
	StreamUserEvents  = "users:events"
	StreamProductEvents = "products:events"
	StreamActivityLog = "activity:log"
)

// EventBuilder provides a fluent interface for building events.
type EventBuilder struct {
	event *Event
}

// NewEventBuilder creates a new event builder.
func NewEventBuilder(eventType string) *EventBuilder {
	return &EventBuilder{
		event: NewEvent(eventType, "", nil),
	}
}

// WithSource sets the event source.
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithPayload sets the event payload.
func (b *EventBuilder) WithPayload(payload map[string]interface{}) *EventBuilder {
	b.event.Payload = payload
	return b
}

// WithPayloadField adds a single field to the payload.
func (b *EventBuilder) WithPayloadField(key string, value interface{}) *EventBuilder {
	if b.event.Payload == nil {
		b.event.Payload = make(map[string]interface{})
	}
	b.event.Payload[key] = value
	return b
}

// WithMetadata sets the event metadata.
func (b *EventBuilder) WithMetadata(metadata map[string]string) *EventBuilder {
	b.event.Metadata = metadata
	return b
}

// WithMetadataField adds a single field to the metadata.
func (b *EventBuilder) WithMetadataField(key, value string) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]string)
	}
	b.event.Metadata[key] = value
	return b
}

// WithCorrelationID sets the correlation ID.
func (b *EventBuilder) WithCorrelationID(id string) *EventBuilder {
	return b.WithMetadataField("correlation_id", id)
}

// WithTraceID sets the trace ID.
func (b *EventBuilder) WithTraceID(id string) *EventBuilder {
	return b.WithMetadataField("trace_id", id)
}

// WithRequestID sets the request ID.
func (b *EventBuilder) WithRequestID(id string) *EventBuilder {
	return b.WithMetadataField("request_id", id)
}

// Build returns the built event.
func (b *EventBuilder) Build() *Event {
	return b.event
}
