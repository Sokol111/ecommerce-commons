package deserialization

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	hambavro "github.com/hamba/avro/v2"
	"github.com/twmb/franz-go/pkg/sr"
)

// Verify at compile time that avroDeserializer implements serde.Deserializer.
var _ serde.Deserializer = (*avroDeserializer)(nil)

type avroDeserializer struct {
	resolver      WriterSchemaResolver
	eventRegistry events.EventRegistry
}

// NewDeserializer creates a new Avro deserializer with Schema Registry integration.
func NewDeserializer(resolver WriterSchemaResolver, eventRegistry events.EventRegistry) serde.Deserializer {
	return &avroDeserializer{
		resolver:      resolver,
		eventRegistry: eventRegistry,
	}
}

func (d *avroDeserializer) Deserialize(data []byte) (events.Event, error) {
	// Parse Confluent wire format to extract schema ID and payload
	var h sr.ConfluentHeader
	schemaID, payload, err := h.DecodeID(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wire format: %w", err)
	}

	// Resolve schema metadata (parsed writer schema and name) by schema ID
	writerSchema, schemaName, err := d.resolver.Resolve(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema for ID %d: %w", schemaID, err)
	}

	// Create new instance of the target event type
	event, err := d.eventRegistry.NewEvent(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to create event for schema %s: %w", schemaName, err)
	}

	// Unmarshal Avro data using writer schema (supports schema evolution)
	if err := hambavro.Unmarshal(writerSchema, payload, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	return event, nil
}
