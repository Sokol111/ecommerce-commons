package serialization

import (
	"context"
	"fmt"
	"sync"

	"github.com/twmb/franz-go/pkg/sr"
)

// SchemaRegistry provides integration with Schema Registry.
// It handles schema registration and ID caching for Kafka message serialization.
// Uses RecordNameStrategy for subject naming: "{schemaName}"
// This allows the same schema to be reused across multiple topics.
type SchemaRegistry interface {
	// GetOrRegisterSchema registers a schema in Schema Registry and returns schema ID.
	// If already registered, returns cached ID.
	// Subject is automatically formed as "{schemaName}" (RecordNameStrategy).
	GetOrRegisterSchema(schemaName string, schemaJSON []byte) (int, error)
}

type schemaRegistry struct {
	client        *sr.Client
	schemaIDCache map[string]int // schemaName -> schema ID
	mu            sync.RWMutex
}

// NewSchemaRegistry creates a new Schema Registry client backed by franz-go/pkg/sr.
func NewSchemaRegistry(client *sr.Client) SchemaRegistry {
	return &schemaRegistry{
		client:        client,
		schemaIDCache: make(map[string]int),
	}
}

func (r *schemaRegistry) GetOrRegisterSchema(schemaName string, schemaJSON []byte) (int, error) {
	// Check if already registered for this schema
	r.mu.RLock()
	cachedID, exists := r.schemaIDCache[schemaName]
	r.mu.RUnlock()

	if exists {
		return cachedID, nil
	}

	// Register in Schema Registry using RecordNameStrategy
	// Subject format: {schemaName} - schema is independent of topic
	// This allows reusing the same schema across multiple topics
	schema := sr.Schema{
		Schema: string(schemaJSON),
		Type:   sr.TypeAvro,
	}

	registeredSchema, err := r.client.CreateSchema(context.Background(), schemaName, schema)
	if err != nil {
		return 0, fmt.Errorf("failed to register schema %s in Schema Registry: %w", schemaName, err)
	}

	// Cache the schema ID by schema name
	r.mu.Lock()
	r.schemaIDCache[schemaName] = registeredSchema.ID
	r.mu.Unlock()

	return registeredSchema.ID, nil
}
