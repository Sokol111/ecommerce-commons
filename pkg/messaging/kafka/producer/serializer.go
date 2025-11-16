package producer

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// Serializer serializes Go structs to Avro bytes with Schema Registry integration
type Serializer interface {
	// Serialize serializes a Go struct to Avro bytes with Schema Registry integration
	//
	// The topic parameter determines the schema subject in Schema Registry.
	// Subject is automatically formed as "{topic}-value" for message values.
	// The schema parameter is the Avro schema definition in JSON format.
	//
	// Returns bytes in format: [0x00][schema_id (4 bytes)][avro_data]
	Serialize(topic string, schema []byte, msg interface{}) ([]byte, error)
}

type schemaCache struct {
	id     int
	schema hambavro.Schema
}

type avroSerializer struct {
	client      schemaregistry.Client
	schemaCache map[string]*schemaCache // subject -> cached schema
	mu          sync.RWMutex
}

// newAvroSerializer creates a new Avro serializer with Schema Registry integration
// Uses hamba/avro for encoding with Schema Registry for schema management
func newAvroSerializer(client schemaregistry.Client) Serializer {
	return &avroSerializer{
		client:      client,
		schemaCache: make(map[string]*schemaCache),
	}
}

func (s *avroSerializer) Serialize(topic string, schemaJSON []byte, msg interface{}) ([]byte, error) {
	subject := topic + "-value"

	// Check cache first
	s.mu.RLock()
	cached, exists := s.schemaCache[subject]
	s.mu.RUnlock()

	var schema hambavro.Schema
	var schemaID int

	if exists {
		// Use cached schema and ID
		schema = cached.schema
		schemaID = cached.id
	} else {
		// Parse Avro schema
		parsedSchema, err := hambavro.Parse(string(schemaJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to parse avro schema: %w", err)
		}
		schema = parsedSchema

		// Register or get schema ID from Schema Registry
		schemaInfo := schemaregistry.SchemaInfo{
			Schema:     string(schemaJSON),
			SchemaType: "AVRO",
		}

		id, err := s.client.Register(subject, schemaInfo, false)
		if err != nil {
			return nil, fmt.Errorf("failed to register schema: %w", err)
		}
		schemaID = id

		// Cache the schema and ID
		s.mu.Lock()
		s.schemaCache[subject] = &schemaCache{
			id:     schemaID,
			schema: schema,
		}
		s.mu.Unlock()
	}

	// Encode message using hamba/avro
	avroData, err := hambavro.Marshal(schema, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal avro data: %w", err)
	}

	// Build Confluent wire format: [0x00][schema_id (4 bytes)][avro_data]
	result := make([]byte, 5+len(avroData))
	result[0] = 0x00 // Magic byte
	result[1] = byte(schemaID >> 24)
	result[2] = byte(schemaID >> 16)
	result[3] = byte(schemaID >> 8)
	result[4] = byte(schemaID)
	copy(result[5:], avroData)

	return result, nil
}
