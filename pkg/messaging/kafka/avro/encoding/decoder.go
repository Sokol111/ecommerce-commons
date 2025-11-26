package encoding

import (
	"fmt"
	"reflect"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	hambavro "github.com/hamba/avro/v2"
)

// Decoder decodes Avro data to Go structs
type Decoder interface {
	// Decode deserializes Avro bytes to a Go object
	// writerSchema is the parsed schema from Schema Registry (used to write the data)
	// schemaName is used to resolve the Go type from TypeMapping (reader schema)
	// This supports Avro schema evolution
	Decode(payload []byte, writerSchema hambavro.Schema, schemaName string) (interface{}, error)
}

type hambaDecoder struct {
	typeMapping *mapping.TypeMapping
}

// NewHambaDecoder creates an Avro decoder using hamba/avro library
func NewHambaDecoder(typeMapping *mapping.TypeMapping) Decoder {
	return &hambaDecoder{
		typeMapping: typeMapping,
	}
}

func (d *hambaDecoder) Decode(payload []byte, writerSchema hambavro.Schema, schemaName string) (interface{}, error) {
	// Resolve Go type from TypeMapping (reader schema)
	binding, err := d.typeMapping.GetBySchemaName(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema binding for %s: %w", schemaName, err)
	}

	// Create new instance of the target type
	// If goType is ProductCreatedEvent, reflect.New creates *ProductCreatedEvent
	targetPtr := reflect.New(binding.GoType)
	target := targetPtr.Interface()

	// Unmarshal Avro data using writer schema (from Registry)
	// Writer schema is already parsed and cached by WriterSchemaResolver
	// This supports schema evolution - writer and reader schemas can differ
	if err := hambavro.Unmarshal(writerSchema, payload, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	// Return pointer to the deserialized object (e.g., *ProductCreatedEvent)
	// This is required because Event interface methods have pointer receivers
	return target, nil
}
