// Package serde provides format-agnostic interfaces for message serialization.
// Implementations for specific formats (Avro, JSON, Protobuf) are in separate packages.
package serde

import "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"

// Serializer serializes events to bytes.
// Implementations may use different formats (Avro, JSON, Protobuf, etc.)
type Serializer interface {
	// Serialize serializes an event to bytes.
	// The event is self-describing: it knows its topic, schema name, and schema bytes.
	//
	// Returns the serialized bytes in implementation-specific format.
	// For Avro with Confluent Schema Registry: [0x00][schema_id (4 bytes)][avro_data]
	Serialize(event events.Event) ([]byte, error)
}

// Deserializer deserializes bytes to events.
// Implementations may use different formats (Avro, JSON, Protobuf, etc.)
type Deserializer interface {
	// Deserialize deserializes bytes to an event.
	//
	// Returns a concrete Go type implementing events.Event.
	Deserialize(data []byte) (events.Event, error)
}
