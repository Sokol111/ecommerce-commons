package deserialization

import (
	"fmt"
	"sync"

	schemaregistry "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// WriterSchemaResolver resolves schema IDs to writer schema metadata from Schema Registry
type WriterSchemaResolver interface {
	// Resolve returns parsed writer schema and schema name for a given schema ID
	// The parsed schema is cached to avoid repeated parsing
	Resolve(schemaID int) (writerSchema hambavro.Schema, schemaName string, err error)
}

type schemaCache struct {
	writerSchema hambavro.Schema
	schemaName   string
}

type registryWriterSchemaResolver struct {
	client schemaregistry.Client
	cache  map[int]*schemaCache
	mu     sync.RWMutex
}

// NewWriterSchemaResolver creates a Schema Registry-based writer schema resolver
func NewWriterSchemaResolver(client schemaregistry.Client) WriterSchemaResolver {
	return &registryWriterSchemaResolver{
		client: client,
		cache:  make(map[int]*schemaCache),
	}
}

func (r *registryWriterSchemaResolver) Resolve(schemaID int) (hambavro.Schema, string, error) {
	// Check cache first
	r.mu.RLock()
	cached, exists := r.cache[schemaID]
	r.mu.RUnlock()

	if exists {
		return cached.writerSchema, cached.schemaName, nil
	}

	// Fetch schema metadata
	writerSchema, schemaName, err := r.fetchSchemaMetadata(schemaID)
	if err != nil {
		return nil, "", err
	}

	// Cache the result
	r.mu.Lock()
	r.cache[schemaID] = &schemaCache{
		writerSchema: writerSchema,
		schemaName:   schemaName,
	}
	r.mu.Unlock()

	return writerSchema, schemaName, nil
}

func (r *registryWriterSchemaResolver) fetchSchemaMetadata(schemaID int) (hambavro.Schema, string, error) {
	// Fetch schema from Schema Registry
	subjectVersions, err := r.client.GetSubjectsAndVersionsByID(schemaID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get subjects for schema ID: %w", err)
	}

	if len(subjectVersions) == 0 {
		return nil, "", fmt.Errorf("no subjects found for schema ID %d", schemaID)
	}

	// Use the first subject (typically there's only one for value schemas)
	subject := subjectVersions[0].Subject

	// Get schema metadata from registry
	schemaMetadata, err := r.client.GetBySubjectAndID(subject, schemaID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch schema from registry: %w", err)
	}

	// Parse schema (writer schema from Registry)
	// This is cached to avoid repeated parsing
	schema, err := hambavro.Parse(schemaMetadata.Schema)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse avro schema: %w", err)
	}

	// Get schema full name (namespace.name)
	namedSchema, ok := schema.(hambavro.NamedSchema)
	if !ok {
		return nil, "", fmt.Errorf("schema is not a NamedSchema (record/enum/fixed)")
	}
	schemaName := namedSchema.FullName()

	return schema, schemaName, nil
}
