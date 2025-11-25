package consumer

import (
	"fmt"
	"reflect"

	hambavro "github.com/hamba/avro/v2"
)

// AvroDecoder decodes Avro data to Go structs
type AvroDecoder interface {
	// Decode deserializes Avro bytes to a Go object based on schema metadata
	Decode(payload []byte, metadata *SchemaMetadata) (interface{}, error)
}

type hambaAvroDecoder struct{}

// newHambaAvroDecoder creates an Avro decoder using hamba/avro library
func newHambaAvroDecoder() AvroDecoder {
	return &hambaAvroDecoder{}
}

func (d *hambaAvroDecoder) Decode(payload []byte, metadata *SchemaMetadata) (interface{}, error) {
	// Create new instance of the target type
	// If goType is ProductCreatedEvent, reflect.New creates *ProductCreatedEvent
	targetPtr := reflect.New(metadata.GoType)
	target := targetPtr.Interface()

	// Unmarshal Avro data into the target
	if err := hambavro.Unmarshal(metadata.Schema, payload, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	// Return pointer to the deserialized object (e.g., *ProductCreatedEvent)
	// This is required because Event interface methods have pointer receivers
	return target, nil
}
