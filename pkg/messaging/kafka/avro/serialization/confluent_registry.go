package serialization

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
)

// SchemaRegistry provides integration with Confluent Schema Registry.
// It handles schema registration and ID caching for Kafka message serialization.
// Uses RecordNameStrategy for subject naming: "{schemaName}"
// This allows the same schema to be reused across multiple topics.
type SchemaRegistry interface {
	// GetOrRegisterSchema registers a schema in Confluent Schema Registry and returns schema ID.
	// If already registered, returns cached ID.
	// Subject is automatically formed as "{schemaName}" (RecordNameStrategy).
	GetOrRegisterSchema(schemaName string, schemaJSON []byte) (int, error)
}

// confluentRegistry implements SchemaRegistry.
type confluentRegistry struct {
	client        schemaregistry.Client
	schemaIDCache map[string]int // schemaName -> schema ID
	mu            sync.RWMutex
}

// NewConfluentRegistry creates a new Confluent Schema Registry client.
func NewConfluentRegistry(client schemaregistry.Client) SchemaRegistry {
	return &confluentRegistry{
		client:        client,
		schemaIDCache: make(map[string]int),
	}
}

func (cr *confluentRegistry) GetOrRegisterSchema(schemaName string, schemaJSON []byte) (int, error) {
	// Check if already registered for this schema
	cr.mu.RLock()
	cachedID, exists := cr.schemaIDCache[schemaName]
	cr.mu.RUnlock()

	if exists {
		return cachedID, nil
	}

	// Register in Confluent Schema Registry using RecordNameStrategy
	// Subject format: {schemaName} - schema is independent of topic
	// This allows reusing the same schema across multiple topics
	schemaInfo := schemaregistry.SchemaInfo{
		Schema:     string(schemaJSON),
		SchemaType: "AVRO",
	}

	id, err := cr.client.Register(schemaName, schemaInfo, false)
	if err != nil {
		return 0, fmt.Errorf("failed to register schema %s in Confluent Schema Registry: %w", schemaName, err)
	}

	// Cache the schema ID by schema name
	cr.mu.Lock()
	cr.schemaIDCache[schemaName] = id
	cr.mu.Unlock()

	return id, nil
}
