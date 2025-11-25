package encoding

import (
	"fmt"
	"reflect"

	hambavro "github.com/hamba/avro/v2"
)

// SchemaMetadata contains schema and type information for deserialization
type SchemaMetadata struct {
	Schema hambavro.Schema
	GoType reflect.Type
}

// Decoder decodes Avro data to Go structs
type Decoder interface {
	// Decode deserializes Avro bytes to a Go object based on schema metadata
	Decode(payload []byte, metadata *SchemaMetadata) (interface{}, error)
}

type hambaDecoder struct{}

// NewHambaDecoder creates an Avro decoder using hamba/avro library
func NewHambaDecoder() Decoder {
	return &hambaDecoder{}
}

func (d *hambaDecoder) Decode(payload []byte, metadata *SchemaMetadata) (interface{}, error) {
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
