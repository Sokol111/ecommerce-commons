// Package events provides common event types and interfaces for messaging.
package events

import "time"

// EventMetadata contains common event metadata used by all domain events.
// Contains technical/observability fields separate from business payload.
type EventMetadata struct {
	// Unique event identifier (UUID).
	EventID string `avro:"event_id" json:"event_id"`
	// Type of the event (e.g., ProductCreated, ProductUpdated).
	EventType string `avro:"event_type" json:"event_type"`
	// Source service that produced the event.
	Source string `avro:"source" json:"source"`
	// Event creation timestamp in milliseconds since epoch.
	Timestamp time.Time `avro:"timestamp" json:"timestamp"`
	// OpenTelemetry trace ID for distributed tracing.
	TraceID *string `avro:"trace_id" json:"trace_id"`
	// Correlation ID for request tracking across services.
	CorrelationID *string `avro:"correlation_id" json:"correlation_id"`
}

// Event is an interface that all event types must implement.
// Generated event types implement this interface automatically.
// Events are self-describing: they know their topic, schema name, and schema bytes.
type Event interface {
	// GetMetadata returns the event metadata (EventID, Timestamp, etc.)
	GetMetadata() *EventMetadata
	// GetTopic returns the Kafka topic for this event type.
	GetTopic() string
	// GetSchemaName returns the full Avro schema name (namespace.name).
	GetSchemaName() string
	// GetSchema returns the Avro schema as JSON bytes.
	GetSchema() []byte
}
