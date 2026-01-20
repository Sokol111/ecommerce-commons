package eventgen

import (
	"encoding/json"
	"fmt"
)

// AvroSchema represents a parsed Avro schema.
type AvroSchema struct {
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Namespace string         `json:"namespace,omitempty"`
	Doc       string         `json:"doc,omitempty"`
	Fields    []AvroField    `json:"fields,omitempty"`
	Aliases   []string       `json:"aliases,omitempty"`
	Default   any            `json:"default,omitempty"`
	Extra     map[string]any `json:"-"` // Additional fields not in the struct
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

// AvroLogicalType represents an Avro logical type.
type AvroLogicalType struct {
	Type        string `json:"type"`
	LogicalType string `json:"logicalType"`
}

// PayloadSchema represents a parsed payload schema with metadata.
type PayloadSchema struct {
	// Original is the original parsed Avro schema
	Original *AvroSchema

	// RawJSON is the raw JSON bytes of the schema
	RawJSON []byte

	// FilePath is the path to the source file
	FilePath string

	// BaseName is the base name without _payload.avsc suffix
	// e.g., "product_created" from "product_created_payload.avsc"
	BaseName string

	// EventName is the derived event name in PascalCase
	// e.g., "ProductCreated" from "product_created"
	EventName string

	// EventTypeName is the full event type name
	// e.g., "ProductCreatedEvent"
	EventTypeName string

	// PayloadTypeName is the payload type name
	// e.g., "ProductCreatedPayload"
	PayloadTypeName string
}

// EnvelopeSchema represents a generated envelope schema (metadata + payload).
type EnvelopeSchema struct {
	// Payload is the source payload schema
	Payload *PayloadSchema

	// Schema is the complete envelope schema
	Schema *AvroSchema

	// SchemaJSON is the envelope schema as JSON bytes
	SchemaJSON []byte

	// CombinedJSON is the schema with EventMetadata inlined (for Avro serialization)
	CombinedJSON []byte
}

// FullName returns the fully qualified schema name (namespace.name).
func (s *AvroSchema) FullName() string {
	if s.Namespace == "" {
		return s.Name
	}
	return s.Namespace + "." + s.Name
}

// Clone creates a deep copy of the schema.
func (s *AvroSchema) Clone() *AvroSchema {
	if s == nil {
		return nil
	}

	clone := &AvroSchema{
		Type:      s.Type,
		Name:      s.Name,
		Namespace: s.Namespace,
		Doc:       s.Doc,
		Default:   s.Default,
	}

	if s.Fields != nil {
		clone.Fields = make([]AvroField, len(s.Fields))
		copy(clone.Fields, s.Fields)
	}

	if s.Aliases != nil {
		clone.Aliases = make([]string, len(s.Aliases))
		copy(clone.Aliases, s.Aliases)
	}

	return clone
}

// MarshalJSON implements custom JSON marshaling for AvroSchema.
func (s *AvroSchema) MarshalJSON() ([]byte, error) {
	type Alias AvroSchema
	return json.Marshal((*Alias)(s))
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

// TopicInfo holds information about a Kafka topic from AsyncAPI.
type TopicInfo struct {
	Name        string
	Description string
	EventNames  []string // Event names from channel messages
}

// GenerationResult holds the result of code generation.
type GenerationResult struct {
	// Payloads are the parsed payload schemas
	Payloads []*PayloadSchema

	// Envelopes are the generated envelope schemas
	Envelopes []*EnvelopeSchema

	// Topics are the topics extracted from AsyncAPI (if provided)
	Topics []*TopicInfo

	// GeneratedFiles is a list of generated file paths
	GeneratedFiles []string
}
