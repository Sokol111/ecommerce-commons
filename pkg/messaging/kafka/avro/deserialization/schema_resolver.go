package deserialization

import (
	"context"
	"fmt"
	"sync"

	hambavro "github.com/hamba/avro/v2"
	"github.com/twmb/franz-go/pkg/sr"
)

// WriterSchemaResolver resolves schema IDs to writer schema metadata from Schema Registry.
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
	client *sr.Client
	cache  map[int]*schemaCache
	mu     sync.RWMutex
}

// NewWriterSchemaResolver creates a Schema Registry-based writer schema resolver.
func NewWriterSchemaResolver(client *sr.Client) WriterSchemaResolver {
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
	// Fetch schema from Schema Registry by ID
	resolvedSchema, err := r.client.SchemaByID(context.Background(), schemaID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch schema for ID %d: %w", schemaID, err)
	}

	// Parse schema (writer schema from Registry)
	schema, err := hambavro.Parse(resolvedSchema.Schema)
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
