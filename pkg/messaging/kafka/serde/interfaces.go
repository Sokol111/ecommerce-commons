// Package serde provides format-agnostic interfaces for message serialization.
// The proto package provides the Protobuf implementation.
package serde

import "google.golang.org/protobuf/proto"

// Serializer serializes proto messages to bytes.
type Serializer interface {
	// Serialize serializes a proto message to bytes.
	Serialize(msg proto.Message) ([]byte, error)
}

// Deserializer deserializes bytes to proto messages.
// The event_type header is used to determine the concrete message type.
type Deserializer interface {
	// Deserialize deserializes bytes to a proto message.
	// headers must contain the "event_type" key with the proto full name.
	Deserialize(data []byte, headers map[string][]byte) (proto.Message, error)
}
