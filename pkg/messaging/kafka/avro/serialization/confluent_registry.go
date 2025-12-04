package serialization

import (
	"fmt"
	"sync"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
)

// ConfluentRegistry provides integration with Confluent Schema Registry.
// It handles schema registration and ID caching for Kafka message serialization.
type ConfluentRegistry interface {
	// RegisterSchema registers a schema in Confluent Schema Registry and returns schema ID
	// Subject is automatically formed as "{binding.Topic}-value"
	RegisterSchema(binding *mapping.SchemaBinding) (int, error)
	// RegisterAllSchemasAtStartup registers all schemas from TypeMapping at application startup
	// Returns error immediately if any schema fails to register (fail-fast approach)
	RegisterAllSchemasAtStartup() error
}

// confluentRegistry implements ConfluentRegistry using TypeMapping
type confluentRegistry struct {
	client        schemaregistry.Client
	typeMapping   *mapping.TypeMapping
	schemaIDCache map[string]int // schemaName -> schema ID
	mu            sync.RWMutex
}

// NewConfluentRegistry creates a new Confluent Schema Registry client using TypeMapping.
func NewConfluentRegistry(client schemaregistry.Client, typeMapping *mapping.TypeMapping) ConfluentRegistry {
	return &confluentRegistry{
		client:        client,
		typeMapping:   typeMapping,
		schemaIDCache: make(map[string]int),
	}
}

func (cr *confluentRegistry) RegisterSchema(binding *mapping.SchemaBinding) (int, error) {
	// Check if already registered for this schema
	cr.mu.RLock()
	cachedID, exists := cr.schemaIDCache[binding.SchemaName]
	cr.mu.RUnlock()

	if exists {
		return cachedID, nil
	}

	// Register in Confluent Schema Registry
	subject := binding.Topic + "-value"
	schemaInfo := schemaregistry.SchemaInfo{
		Schema:     string(binding.SchemaJSON),
		SchemaType: "AVRO",
	}

	id, err := cr.client.Register(subject, schemaInfo, false)
	if err != nil {
		return 0, fmt.Errorf("failed to register schema %s in Confluent Schema Registry: %w", binding.SchemaName, err)
	}

	// Cache the schema ID by schema name
	cr.mu.Lock()
	cr.schemaIDCache[binding.SchemaName] = id
	cr.mu.Unlock()

	return id, nil
}

// RegisterAllSchemasAtStartup registers all schemas from the type mapping in Confluent Schema Registry.
// This should be called once at application startup to ensure all schemas are registered.
// Returns error immediately if any schema fails to register (fail-fast approach).
func (cr *confluentRegistry) RegisterAllSchemasAtStartup() error {
	allBindings := cr.typeMapping.GetAllBindings()

	for _, binding := range allBindings {
		_, err := cr.RegisterSchema(binding)
		if err != nil {
			return fmt.Errorf("failed to register schema %s for topic %s: %w", binding.SchemaName, binding.Topic, err)
		}
	}

	return nil
}
