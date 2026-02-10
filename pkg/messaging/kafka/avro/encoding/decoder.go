package encoding

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	hambavro "github.com/hamba/avro/v2"
)

// Decoder decodes Avro data to Go structs.
type Decoder interface {
	// Decode deserializes Avro bytes to an Event.
	// writerSchema is the parsed schema from Schema Registry (used to write the data).
	// schemaName is used to resolve the Go type from EventRegistry.
	// This supports Avro schema evolution.
	Decode(payload []byte, writerSchema hambavro.Schema, schemaName string) (events.Event, error)
}

type hambaDecoder struct {
	eventRegistry events.EventRegistry
}

// NewHambaDecoder creates an Avro decoder using hamba/avro library.
func NewHambaDecoder(eventRegistry events.EventRegistry) Decoder {
	return &hambaDecoder{
		eventRegistry: eventRegistry,
	}
}

func (d *hambaDecoder) Decode(payload []byte, writerSchema hambavro.Schema, schemaName string) (events.Event, error) {
	// Create new instance of the target event type from EventRegistry
	event, err := d.eventRegistry.NewEvent(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to create event for schema %s: %w", schemaName, err)
	}

	// Unmarshal Avro data using writer schema (from Registry)
	// Writer schema is already parsed and cached by WriterSchemaResolver
	// This supports schema evolution - writer and reader schemas can differ
	if err := hambavro.Unmarshal(writerSchema, payload, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	// Return the deserialized event
	return event, nil
}
