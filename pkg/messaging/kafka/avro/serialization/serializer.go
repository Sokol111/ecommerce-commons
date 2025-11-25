package serialization

import (
	"fmt"
	"reflect"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
)

// Serializer serializes Go structs to Avro bytes with Schema Registry integration
type Serializer interface {
	// Serialize serializes a Go struct to Avro bytes with Schema Registry integration
	//
	// The topic parameter determines the schema subject in Schema Registry.
	// Subject is automatically formed as "{topic}-value" for message values.
	// The msg parameter must be a type registered in TypeMapping.
	//
	// Returns bytes in format: [0x00][schema_id (4 bytes)][avro_data]
	Serialize(topic string, msg interface{}) ([]byte, error)
}

type avroSerializer struct {
	registry SchemaRegistry
	encoder  encoding.Encoder
	builder  encoding.WireFormatBuilder
}

// NewAvroSerializer creates a new Avro serializer with Schema Registry integration
// Uses composition of specialized components for separation of concerns
func NewAvroSerializer(registry SchemaRegistry, encoder encoding.Encoder, builder encoding.WireFormatBuilder) Serializer {
	return &avroSerializer{
		registry: registry,
		encoder:  encoder,
		builder:  builder,
	}
}

func (s *avroSerializer) Serialize(topic string, msg interface{}) ([]byte, error) {
	// Get Go type from message
	msgType := reflect.TypeOf(msg)
	if msgType.Kind() == reflect.Ptr {
		msgType = msgType.Elem()
	}

	// Get schema from registry by type
	schema, err := s.registry.GetSchemaByType(msgType)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for type %s: %w", msgType, err)
	}

	// Register or get schema ID from Schema Registry
	schemaID, err := s.registry.RegisterSchema(topic+"-value", schema.String())
	if err != nil {
		return nil, fmt.Errorf("failed to register schema: %w", err)
	}

	// Encode message using encoder
	avroData, err := s.encoder.Encode(msg, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to encode avro data: %w", err)
	}

	// Build Confluent wire format
	return s.builder.Build(schemaID, avroData), nil
}
