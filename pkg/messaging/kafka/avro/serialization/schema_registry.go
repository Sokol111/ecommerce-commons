package serialization

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// SchemaMapping maps Go types to their Avro schemas (JSON format)
type SchemaMapping map[reflect.Type][]byte

// SchemaRegistry provides schema lookup and registration
type SchemaRegistry interface {
	// GetSchemaByType returns schema for a given Go type
	GetSchemaByType(goType reflect.Type) (hambavro.Schema, error)
	// RegisterSchema registers schema in Schema Registry and returns schema ID
	RegisterSchema(subject string, schemaJSON string) (int, error)
}

type schemaCacheKey struct {
	subject    string
	schemaJSON string
}

// typeSchemaRegistry implements SchemaRegistry using SchemaMapping
type typeSchemaRegistry struct {
	client            schemaregistry.Client
	schemaMapping     SchemaMapping
	parsedSchemaCache map[reflect.Type]hambavro.Schema
	schemaIDCache     map[schemaCacheKey]int // (subject, schema) -> schema ID
	mu                sync.RWMutex
}

// NewTypeSchemaRegistry creates a new schema registry based on schema mapping
func NewTypeSchemaRegistry(client schemaregistry.Client, schemaMap SchemaMapping) SchemaRegistry {
	return &typeSchemaRegistry{
		client:            client,
		schemaMapping:     schemaMap,
		parsedSchemaCache: make(map[reflect.Type]hambavro.Schema),
		schemaIDCache:     make(map[schemaCacheKey]int),
	}
}

func (r *typeSchemaRegistry) GetSchemaByType(goType reflect.Type) (hambavro.Schema, error) {
	// Check parsed schema cache
	r.mu.RLock()
	cached, exists := r.parsedSchemaCache[goType]
	r.mu.RUnlock()

	if exists {
		return cached, nil
	}

	// Get schema JSON
	schemaJSON, ok := r.schemaMapping[goType]
	if !ok {
		return nil, fmt.Errorf("no schema registered for Go type: %s", goType)
	}

	// Parse Avro schema
	schema, err := hambavro.Parse(string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema for type %s: %w", goType, err)
	}

	// Cache parsed schema
	r.mu.Lock()
	r.parsedSchemaCache[goType] = schema
	r.mu.Unlock()

	return schema, nil
}

func (r *typeSchemaRegistry) RegisterSchema(subject string, schemaJSON string) (int, error) {
	// Check if already registered for this subject+schema combination
	key := schemaCacheKey{subject: subject, schemaJSON: schemaJSON}
	r.mu.RLock()
	cachedID, exists := r.schemaIDCache[key]
	r.mu.RUnlock()

	if exists {
		return cachedID, nil
	}

	// Register in Schema Registry
	schemaInfo := schemaregistry.SchemaInfo{
		Schema:     schemaJSON,
		SchemaType: "AVRO",
	}

	id, err := r.client.Register(subject, schemaInfo, false)
	if err != nil {
		return 0, fmt.Errorf("failed to register schema in registry: %w", err)
	}

	// Cache the schema ID
	r.mu.Lock()
	r.schemaIDCache[key] = id
	r.mu.Unlock()

	return id, nil
}
