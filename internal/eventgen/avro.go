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

// PayloadSchema represents a parsed payload schema with metadata.
type PayloadSchema struct {
	// Original is the original parsed Avro schema
	Original *AvroSchema

	// FilePath is the path to the source file
	FilePath string

	// BaseName is the base name without _payload.avsc suffix
	// e.g., "product_created" from "product_created_payload.avsc"
	BaseName string

	// Topic is the Kafka topic for this event (from schema "topic" field)
	Topic string

	// CombinedJSON is the envelope schema with EventMetadata inlined (for Avro serialization)
	// Populated by Parser.CreateEnvelope()
	CombinedJSON []byte
}

// EventName returns the event name in PascalCase (e.g., "ProductCreated").
func (p *PayloadSchema) EventName() string {
	return toPascalCase(p.BaseName)
}

// EventTypeName returns the full event type name (e.g., "ProductCreatedEvent").
func (p *PayloadSchema) EventTypeName() string {
	return p.EventName() + "Event"
}

// PayloadTypeName returns the payload type name (e.g., "ProductCreatedPayload").
func (p *PayloadSchema) PayloadTypeName() string {
	return p.EventName() + "Payload"
}

// SchemaFullName returns the full qualified schema name (namespace.EventTypeName).
// e.g., "com.ecommerce.events.ProductCreatedEvent"
func (p *PayloadSchema) SchemaFullName() string {
	if p.Original.Namespace == "" {
		return p.EventTypeName()
	}
	return p.Original.Namespace + "." + p.EventTypeName()
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

	if schema.Type != "record" {
		return nil, fmt.Errorf("expected record type, got %q", schema.Type)
	}

	if schema.Name == "" {
		return nil, fmt.Errorf("schema name is required")
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
