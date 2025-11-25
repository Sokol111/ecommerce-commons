package deserialization

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
)

// Deserializer deserializes Avro bytes to Go structs using Schema Registry
type Deserializer interface {
	// Deserialize deserializes Avro bytes to a Go struct
	//
	// The data must be in format: [0x00][schema_id (4 bytes)][avro_data]
	//
	// Returns a concrete Go type based on the type registry configuration.
	Deserialize(data []byte) (interface{}, error)
}

type avroDeserializer struct {
	parser   encoding.WireFormatParser
	resolver SchemaResolver
	decoder  encoding.Decoder
}

// NewDeserializer creates a new Avro deserializer with Schema Registry integration
// Uses composition of specialized components for separation of concerns
func NewDeserializer(parser encoding.WireFormatParser, resolver SchemaResolver, decoder encoding.Decoder) Deserializer {
	return &avroDeserializer{
		parser:   parser,
		resolver: resolver,
		decoder:  decoder,
	}
}

func (d *avroDeserializer) Deserialize(data []byte) (interface{}, error) {
	// Parse wire format to extract schema ID and payload
	schemaID, payload, err := d.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wire format: %w", err)
	}

	// Resolve schema metadata
	metadata, err := d.resolver.Resolve(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema for ID %d: %w", schemaID, err)
	}

	// Decode Avro payload
	result, err := d.decoder.Decode(payload, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode avro data: %w", err)
	}

	return result, nil
}
