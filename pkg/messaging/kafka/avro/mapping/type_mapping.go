package mapping

import (
	"fmt"
	"reflect"

	hambavro "github.com/hamba/avro/v2"
)

// SchemaBinding contains the binding between Go type, Avro schema, and Kafka topic.
type SchemaBinding struct {
	// GoType is the Go reflect.Type for serialization/deserialization
	GoType reflect.Type
	// SchemaJSON is the Avro schema in JSON format
	SchemaJSON []byte
	// SchemaName is the full name of the Avro schema (namespace.name)
	SchemaName string
	// Topic is the Kafka topic this schema is associated with
	Topic string
	// ParsedSchema is the parsed Avro schema (cached for performance)
	ParsedSchema hambavro.Schema
}

// TypeMapping is a local registry for mapping between Go types, Avro schemas, and Kafka topics.
type TypeMapping struct {
	// typeToBinding maps Go types to schema bindings (for serialization)
	typeToBinding map[reflect.Type]*SchemaBinding
	// nameToBinding maps schema names to schema bindings (for deserialization)
	nameToBinding map[string]*SchemaBinding
}

// NewTypeMapping creates a new type mapping registry.
func NewTypeMapping() *TypeMapping {
	return &TypeMapping{
		typeToBinding: make(map[reflect.Type]*SchemaBinding),
		nameToBinding: make(map[string]*SchemaBinding),
	}
}

// Register adds a schema binding to the type mapping.
// This registers the binding for both serialization (by Go type) and deserialization (by schema name).
func (tm *TypeMapping) Register(goType reflect.Type, schemaJSON []byte, schemaName string, topic string) error {
	if goType == nil {
		return fmt.Errorf("goType cannot be nil")
	}
	if len(schemaJSON) == 0 {
		return fmt.Errorf("schemaJSON cannot be empty")
	}
	if schemaName == "" {
		return fmt.Errorf("schemaName cannot be empty")
	}
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// Parse Avro schema immediately to validate and cache it
	parsedSchema, err := hambavro.Parse(string(schemaJSON))
	if err != nil {
		return fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	binding := &SchemaBinding{
		GoType:       goType,
		SchemaJSON:   schemaJSON,
		SchemaName:   schemaName,
		Topic:        topic,
		ParsedSchema: parsedSchema,
	}

	tm.typeToBinding[goType] = binding
	tm.nameToBinding[schemaName] = binding

	return nil
}

// GetByType returns schema binding by Go type (used for serialization).
func (tm *TypeMapping) GetByType(goType reflect.Type) (*SchemaBinding, error) {
	binding, ok := tm.typeToBinding[goType]
	if !ok {
		return nil, fmt.Errorf("no schema registered for Go type: %s", goType)
	}
	return binding, nil
}

// GetByValue returns schema binding by value instance.
// Automatically handles pointer types by extracting the underlying type.
func (tm *TypeMapping) GetByValue(value interface{}) (*SchemaBinding, error) {
	goType := reflect.TypeOf(value)
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	return tm.GetByType(goType)
}

// GetBySchemaName returns schema binding by Avro schema name (used for deserialization).
func (tm *TypeMapping) GetBySchemaName(schemaName string) (*SchemaBinding, error) {
	binding, ok := tm.nameToBinding[schemaName]
	if !ok {
		return nil, fmt.Errorf("no schema registered for schema name: %s", schemaName)
	}
	return binding, nil
}

// GetAllBindings returns a slice of all registered schema bindings.
func (tm *TypeMapping) GetAllBindings() []*SchemaBinding {
	bindings := make([]*SchemaBinding, 0, len(tm.nameToBinding))
	for _, binding := range tm.nameToBinding {
		bindings = append(bindings, binding)
	}
	return bindings
}
