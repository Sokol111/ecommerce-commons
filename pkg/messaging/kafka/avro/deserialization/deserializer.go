package deserialization

import (
	"encoding/binary"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	hambavro "github.com/hamba/avro/v2"
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
	schemaID, payload, err := parseConfluentWireFormat(data)
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

// parseConfluentWireFormat extracts schema ID and payload from Confluent wire format.
// Format: [0x00][schema_id (4 bytes big-endian)][payload]
func parseConfluentWireFormat(data []byte) (schemaID int, payload []byte, err error) {
	if len(data) < 5 {
		return 0, nil, fmt.Errorf("data too short: expected at least 5 bytes, got %d", len(data))
	}

	if data[0] != 0x00 {
		return 0, nil, fmt.Errorf("invalid magic byte: expected 0x00, got 0x%02x", data[0])
	}

	schemaID = int(binary.BigEndian.Uint32(data[1:5]))
	return schemaID, data[5:], nil
}
