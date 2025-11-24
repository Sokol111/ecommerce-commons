package consumer

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hambavro "github.com/hamba/avro/v2"
)

// Deserializer deserializes Avro bytes to Go structs using Schema Registry
type Deserializer interface {
	// Deserialize deserializes Avro bytes to a Go struct
	//
	// The data must be in format: [0x00][schema_id (4 bytes)][avro_data]
	//
	// Returns a concrete Go type based on the type registry configuration.
	Deserialize(data []byte) (interface{}, error)
}

// typeMapping maps Avro schema full names to Go types
type typeMapping map[string]reflect.Type

type schemaInfo struct {
	schema     hambavro.Schema
	schemaName string
	goType     reflect.Type
}

type avroDeserializer struct {
	client      schemaregistry.Client
	typeMapping typeMapping
	schemaCache map[int]*schemaInfo // schema ID -> schema info
	mu          sync.RWMutex
}

// newAvroDeserializer creates a new Avro deserializer with Schema Registry integration
// Uses hamba/avro for decoding with Schema Registry for schema management
func newAvroDeserializer(client schemaregistry.Client, typeMap typeMapping) Deserializer {
	return &avroDeserializer{
		client:      client,
		typeMapping: typeMap,
		schemaCache: make(map[int]*schemaInfo),
	}
}

func (d *avroDeserializer) Deserialize(data []byte) (interface{}, error) {
	// Validate Confluent wire format
	if len(data) < 5 {
		return nil, fmt.Errorf("data too short: expected at least 5 bytes, got %d", len(data))
	}

	// Check magic byte
	if data[0] != 0x00 {
		return nil, fmt.Errorf("invalid magic byte: expected 0x00, got 0x%02x", data[0])
	}

	// Extract schema ID (big-endian)
	schemaID := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	// Get schema info (cached or fetch from registry)
	info, err := d.getSchemaInfo(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema info for ID %d: %w", schemaID, err)
	}

	// Create new instance of the target type
	// If goType is ProductCreatedEvent, reflect.New creates *ProductCreatedEvent
	targetPtr := reflect.New(info.goType)
	target := targetPtr.Interface()

	// Unmarshal Avro data into the target
	if err := hambavro.Unmarshal(info.schema, data[5:], target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avro data: %w", err)
	}

	// Return pointer to the deserialized object (e.g., *ProductCreatedEvent)
	// This is required because Event interface methods have pointer receivers
	return target, nil
}

func (d *avroDeserializer) getSchemaInfo(schemaID int) (*schemaInfo, error) {
	// Check cache first
	d.mu.RLock()
	cached, exists := d.schemaCache[schemaID]
	d.mu.RUnlock()

	if exists {
		return cached, nil
	}

	// Fetch schema from Schema Registry
	// First, get subject info for this schema ID
	subjectVersions, err := d.client.GetSubjectsAndVersionsByID(schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subjects for schema ID: %w", err)
	}

	if len(subjectVersions) == 0 {
		return nil, fmt.Errorf("no subjects found for schema ID %d", schemaID)
	}

	// Use the first subject (typically there's only one for value schemas)
	subject := subjectVersions[0].Subject

	// Get schema metadata
	schemaMetadata, err := d.client.GetBySubjectAndID(subject, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from registry: %w", err)
	}

	// Parse Avro schema
	schema, err := hambavro.Parse(schemaMetadata.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avro schema: %w", err)
	}

	// Get schema full name (namespace.name)
	// Event schemas are always NamedSchema (record type)
	namedSchema, ok := schema.(hambavro.NamedSchema)
	if !ok {
		return nil, fmt.Errorf("schema is not a NamedSchema (record/enum/fixed)")
	}
	schemaName := namedSchema.FullName()

	// Find matching Go type
	goType, ok := d.typeMapping[schemaName]
	if !ok {
		// Log all registered types for debugging
		registeredTypes := make([]string, 0, len(d.typeMapping))
		for name := range d.typeMapping {
			registeredTypes = append(registeredTypes, name)
		}
		return nil, fmt.Errorf("no Go type registered for schema: %s, registered types: %v", schemaName, registeredTypes)
	}

	// Cache the schema info
	info := &schemaInfo{
		schema:     schema,
		schemaName: schemaName,
		goType:     goType,
	}

	d.mu.Lock()
	d.schemaCache[schemaID] = info
	d.mu.Unlock()

	return info, nil
}
