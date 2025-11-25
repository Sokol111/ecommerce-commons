package deserialization

import (
	"fmt"
	"reflect"
	"sync"

	schemaregistry "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// TypeMapping maps Avro schema full names to Go types
type TypeMapping map[string]reflect.Type

// SchemaResolver resolves schema IDs to schema metadata
type SchemaResolver interface {
	// Resolve returns schema and Go type for a given schema ID
	Resolve(schemaID int) (schema hambavro.Schema, goType reflect.Type, err error)
}

type schemaCache struct {
	schema hambavro.Schema
	goType reflect.Type
}

type registrySchemaResolver struct {
	client      schemaregistry.Client
	typeMapping TypeMapping
	cache       map[int]*schemaCache
	mu          sync.RWMutex
}

// NewRegistrySchemaResolver creates a Schema Registry-based resolver
func NewRegistrySchemaResolver(client schemaregistry.Client, typeMap TypeMapping) SchemaResolver {
	return &registrySchemaResolver{
		client:      client,
		typeMapping: typeMap,
		cache:       make(map[int]*schemaCache),
	}
}

func (r *registrySchemaResolver) Resolve(schemaID int) (hambavro.Schema, reflect.Type, error) {
	// Check cache first
	r.mu.RLock()
	cached, exists := r.cache[schemaID]
	r.mu.RUnlock()

	if exists {
		return cached.schema, cached.goType, nil
	}

	// Fetch and parse schema
	schema, goType, err := r.fetchAndParseSchema(schemaID)
	if err != nil {
		return nil, nil, err
	}

	// Cache the result
	r.mu.Lock()
	r.cache[schemaID] = &schemaCache{
		schema: schema,
		goType: goType,
	}
	r.mu.Unlock()

	return schema, goType, nil
}

func (r *registrySchemaResolver) fetchAndParseSchema(schemaID int) (hambavro.Schema, reflect.Type, error) {
	// Fetch schema from Schema Registry
	subjectVersions, err := r.client.GetSubjectsAndVersionsByID(schemaID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get subjects for schema ID: %w", err)
	}

	if len(subjectVersions) == 0 {
		return nil, nil, fmt.Errorf("no subjects found for schema ID %d", schemaID)
	}

	// Use the first subject (typically there's only one for value schemas)
	subject := subjectVersions[0].Subject

	// Get schema metadata from registry
	schemaMetadata, err := r.client.GetBySubjectAndID(subject, schemaID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch schema from registry: %w", err)
	}

	// Parse Avro schema
	schema, err := hambavro.Parse(schemaMetadata.Schema)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse avro schema: %w", err)
	}

	// Get schema full name (namespace.name)
	namedSchema, ok := schema.(hambavro.NamedSchema)
	if !ok {
		return nil, nil, fmt.Errorf("schema is not a NamedSchema (record/enum/fixed)")
	}
	schemaName := namedSchema.FullName()

	// Find matching Go type
	goType, ok := r.typeMapping[schemaName]
	if !ok {
		// Log all registered types for debugging
		registeredTypes := make([]string, 0, len(r.typeMapping))
		for name := range r.typeMapping {
			registeredTypes = append(registeredTypes, name)
		}
		return nil, nil, fmt.Errorf("no Go type registered for schema: %s, registered types: %v", schemaName, registeredTypes)
	}

	return schema, goType, nil
}
