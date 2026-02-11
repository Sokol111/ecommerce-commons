package serialization

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	hambavro "github.com/hamba/avro/v2"
)

// Verify at compile time that avroSerializer implements serde.Serializer.
var _ serde.Serializer = (*avroSerializer)(nil)

type avroSerializer struct {
	schemaRegistry SchemaRegistry
	schemaCache    map[string]hambavro.Schema // schemaName -> parsed schema
	mu             sync.RWMutex
}

// NewSerializer creates a new Avro serializer with Confluent Schema Registry integration.
// Events are self-describing: they know their schema name and schema bytes.
func NewSerializer(schemaRegistry SchemaRegistry) serde.Serializer {
	return &avroSerializer{
		schemaRegistry: schemaRegistry,
		schemaCache:    make(map[string]hambavro.Schema),
	}
}

func (s *avroSerializer) Serialize(event events.Event) ([]byte, error) {
	schemaName := event.GetSchemaName()
	schemaJSON := event.GetSchema()

	// Register or get schema ID from Confluent Schema Registry
	schemaID, err := s.schemaRegistry.GetOrRegisterSchema(schemaName, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to register schema in Confluent: %w", err)
	}

	// Get or parse schema (cached)
	parsedSchema, err := s.getParsedSchema(schemaName, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Encode message to Avro bytes
	avroData, err := hambavro.Marshal(parsedSchema, event)
	if err != nil {
		return nil, fmt.Errorf("failed to encode avro data: %w", err)
	}

	// Build Confluent wire format: [0x00][schema_id (4 bytes)][avro_data]
	return buildConfluentWireFormat(schemaID, avroData), nil
}

// buildConfluentWireFormat creates Confluent wire format from schema ID and payload.
// Format: [0x00][schema_id (4 bytes big-endian)][payload].
func buildConfluentWireFormat(schemaID int, payload []byte) []byte {
	result := make([]byte, 5+len(payload))
	result[0] = 0x00                                          // Magic byte
	binary.BigEndian.PutUint32(result[1:5], uint32(schemaID)) //nolint:gosec // schemaID is always positive from schema registry
	copy(result[5:], payload)
	return result
}

func (s *avroSerializer) getParsedSchema(schemaName string, schemaJSON []byte) (hambavro.Schema, error) {
	// Check cache first
	s.mu.RLock()
	cached, exists := s.schemaCache[schemaName]
	s.mu.RUnlock()

	if exists {
		return cached, nil
	}

	// Parse schema
	parsed, err := hambavro.Parse(string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro schema %s: %w", schemaName, err)
	}

	// Cache it
	s.mu.Lock()
	s.schemaCache[schemaName] = parsed
	s.mu.Unlock()

	return parsed, nil
}
