package encoding

import (
	"fmt"
	"reflect"

	hambavro "github.com/hamba/avro/v2"
)

// Decoder decodes Avro data to Go structs
type Decoder interface {
	// Decode deserializes Avro bytes to a Go object
	Decode(payload []byte, schema hambavro.Schema, goType reflect.Type) (interface{}, error)
}

type hambaDecoder struct{}

// NewHambaDecoder creates an Avro decoder using hamba/avro library
func NewHambaDecoder() Decoder {
	return &hambaDecoder{}
}

func (d *hambaDecoder) Decode(payload []byte, schema hambavro.Schema, goType reflect.Type) (interface{}, error) {
	// Create new instance of the target type
	// If goType is ProductCreatedEvent, reflect.New creates *ProductCreatedEvent
	targetPtr := reflect.New(goType)
	target := targetPtr.Interface()

	// Unmarshal Avro data into the target
	if err := hambavro.Unmarshal(schema, payload, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	// Return pointer to the deserialized object (e.g., *ProductCreatedEvent)
	// This is required because Event interface methods have pointer receivers
	return target, nil
}
