package serialization

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
)

// Serializer serializes Go structs to Avro bytes with Confluent Schema Registry integration
type Serializer interface {
	// Serialize serializes a Go struct to Avro bytes using topic from schema binding
	// The msg parameter must be a type registered in TypeMapping with topic configured
	//
	// Returns bytes in format: [0x00][schema_id (4 bytes)][avro_data]
	Serialize(msg interface{}) ([]byte, error)
}

type serializer struct {
	typeMapping       *mapping.TypeMapping
	confluentRegistry ConfluentRegistry
	encoder           encoding.Encoder
	builder           encoding.WireFormatBuilder
}

// NewSerializer creates a new Avro serializer with Confluent Schema Registry integration
// Uses composition of specialized components for separation of concerns
func NewSerializer(
	typeMapping *mapping.TypeMapping,
	confluentRegistry ConfluentRegistry,
	encoder encoding.Encoder,
	builder encoding.WireFormatBuilder,
) Serializer {
	return &serializer{
		typeMapping:       typeMapping,
		confluentRegistry: confluentRegistry,
		encoder:           encoder,
		builder:           builder,
	}
}

func (s *serializer) Serialize(msg interface{}) ([]byte, error) {
	// Get schema binding directly from type mapping
	binding, err := s.typeMapping.GetByValue(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema binding: %w", err)
	}

	// Register or get schema ID from Confluent Schema Registry
	schemaID, err := s.confluentRegistry.RegisterSchema(binding)
	if err != nil {
		return nil, fmt.Errorf("failed to register schema in Confluent: %w", err)
	}

	// Encode message using encoder with cached parsed schema
	avroData, err := s.encoder.Encode(msg, binding.ParsedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to encode avro data: %w", err)
	}

	// Build Confluent wire format
	return s.builder.Build(schemaID, avroData), nil
}
