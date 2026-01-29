package eventgen

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AvroSchema represents a parsed Avro schema.
type AvroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Doc       string      `json:"doc,omitempty"`
	Topic     string      `json:"topic,omitempty"` // Kafka topic for this event
	Fields    []AvroField `json:"fields,omitempty"`
	Aliases   []string    `json:"aliases,omitempty"`
	Default   any         `json:"default,omitempty"`
}

// AvroField represents a field in an Avro record.
type AvroField struct {
	Name    string   `json:"name"`
	Type    any      `json:"type"` // Can be string, array, or object
	Doc     string   `json:"doc,omitempty"`
	Default any      `json:"default,omitempty"`
	Order   string   `json:"order,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

// EventName returns the event name in PascalCase (e.g., "ProductCreated").
// Derived from Name by removing "Payload" suffix.
func (s *AvroSchema) EventName() string {
	return strings.TrimSuffix(s.Name, "Payload")
}

// BaseName returns the event name in snake_case (e.g., "product_created").
// Used for schema file names.
func (s *AvroSchema) BaseName() string {
	return toSnakeCase(s.EventName())
}

// EventTypeName returns the full event type name (e.g., "ProductCreatedEvent").
func (s *AvroSchema) EventTypeName() string {
	return s.EventName() + "Event"
}

// PayloadTypeName returns the payload type name (same as schema Name, e.g., "ProductCreatedPayload").
func (s *AvroSchema) PayloadTypeName() string {
	return s.Name
}

// EventSchemaFullName returns the full qualified schema name for event (namespace.EventTypeName).
// e.g., "com.ecommerce.events.ProductCreatedEvent"
func (s *AvroSchema) EventSchemaFullName() string {
	if s.Namespace == "" {
		return s.EventTypeName()
	}
	return s.Namespace + "." + s.EventTypeName()
}

// FullName returns the fully qualified schema name (namespace.name).
func (s *AvroSchema) FullName() string {
	if s.Namespace == "" {
		return s.Name
	}
	return s.Namespace + "." + s.Name
}

// ParseAvroSchema parses JSON bytes into an AvroSchema.
func ParseAvroSchema(data []byte) (*AvroSchema, error) {
	var schema AvroSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse Avro schema: %w", err)
	}
	if schema.Topic == "" {
		return nil, fmt.Errorf("schema %q is missing required 'topic' field", schema.Name)
	}
	return &schema, nil
}

// ToJSON converts the schema to formatted JSON bytes.
func (s *AvroSchema) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// toPascalCase converts snake_case to PascalCase.
// e.g., "product_created" -> "ProductCreated"
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// toSnakeCase converts PascalCase to snake_case.
// e.g., "ProductCreated" -> "product_created"
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
