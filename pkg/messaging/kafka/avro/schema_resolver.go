package avro

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// SchemaMetadata contains schema and type information for deserialization
type SchemaMetadata struct {
	Schema     hambavro.Schema
	SchemaName string
	GoType     reflect.Type
}

// SchemaResolver resolves schema IDs to schema metadata
type SchemaResolver interface {
	// Resolve returns schema metadata for a given schema ID
	Resolve(schemaID int) (*SchemaMetadata, error)
}

type registrySchemaResolver struct {
	client      schemaregistry.Client
	typeMapping TypeMapping
	cache       map[int]*SchemaMetadata
	mu          sync.RWMutex
}

// NewRegistrySchemaResolver creates a Schema Registry-based resolver
func NewRegistrySchemaResolver(client schemaregistry.Client, typeMap TypeMapping) SchemaResolver {
	return &registrySchemaResolver{
		client:      client,
		typeMapping: typeMap,
		cache:       make(map[int]*SchemaMetadata),
	}
}

func (r *registrySchemaResolver) Resolve(schemaID int) (*SchemaMetadata, error) {
	// Check cache first
	r.mu.RLock()
	cached, exists := r.cache[schemaID]
	r.mu.RUnlock()

	if exists {
		return cached, nil
	}

	// Fetch and parse schema
	metadata, err := r.fetchAndParseSchema(schemaID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.mu.Lock()
	r.cache[schemaID] = metadata
	r.mu.Unlock()

	return metadata, nil
}

func (r *registrySchemaResolver) fetchAndParseSchema(schemaID int) (*SchemaMetadata, error) {
	// Fetch schema from Schema Registry
	subjectVersions, err := r.client.GetSubjectsAndVersionsByID(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subjects for schema ID: %w", err)
	}

	if len(subjectVersions) == 0 {
		return nil, fmt.Errorf("no subjects found for schema ID %d", schemaID)
	}

	// Use the first subject (typically there's only one for value schemas)
	subject := subjectVersions[0].Subject

	// Get schema metadata from registry
	schemaMetadata, err := r.client.GetBySubjectAndID(subject, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from registry: %w", err)
	}

	// Parse Avro schema
	schema, err := hambavro.Parse(schemaMetadata.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro schema: %w", err)
	}

	// Get schema full name (namespace.name)
	namedSchema, ok := schema.(hambavro.NamedSchema)
	if !ok {
		return nil, fmt.Errorf("schema is not a NamedSchema (record/enum/fixed)")
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
		return nil, fmt.Errorf("no Go type registered for schema: %s, registered types: %v", schemaName, registeredTypes)
	}

	return &SchemaMetadata{
		Schema:     schema,
		SchemaName: schemaName,
		GoType:     goType,
	}, nil
}
