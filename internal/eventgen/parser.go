package eventgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hamba/avro/v2"
)

// Parser handles parsing of Avro payload schemas.
type Parser struct {
	config   *Config
	metadata *AvroSchema
}

// NewParser creates a new schema parser.
func NewParser(cfg *Config, metadata *AvroSchema) *Parser {
	return &Parser{
		config:   cfg,
		metadata: metadata,
	}
}

// ParsePayloads reads and parses all *_payload.avsc files from the payloads directory.
func (p *Parser) ParsePayloads() ([]*PayloadSchema, error) {
	pattern := filepath.Join(p.config.PayloadsDir, "*_payload.avsc")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob payload files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *_payload.avsc files found in %s", p.config.PayloadsDir)
	}

	payloads := make([]*PayloadSchema, 0, len(files))
	for _, file := range files {
		payload, err := p.parsePayloadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		payloads = append(payloads, payload)
	}

	return payloads, nil
}

// parsePayloadFile parses a single payload schema file.
func (p *Parser) parsePayloadFile(filePath string) (*PayloadSchema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	schema, err := ParseAvroSchema(data)
	if err != nil {
		return nil, err
	}

	baseName := extractBaseName(filePath)
	eventName := toPascalCase(baseName)

	payload := &PayloadSchema{
		Original:        schema,
		RawJSON:         data,
		FilePath:        filePath,
		BaseName:        baseName,
		EventName:       eventName,
		EventTypeName:   eventName + "Event",
		PayloadTypeName: eventName + "Payload",
	}

	// Validate that the schema name matches expected payload name
	expectedName := payload.PayloadTypeName
	if schema.Name != expectedName {
		// Allow the schema name to be anything, but warn if it doesn't match
		if p.config.Verbose {
			fmt.Printf("Note: Schema name %q doesn't match expected %q (derived from filename)\n",
				schema.Name, expectedName)
		}
		// Use the actual schema name
		payload.PayloadTypeName = schema.Name
	}

	return payload, nil
}

// CreateEnvelope creates an envelope schema from a payload schema.
func (p *Parser) CreateEnvelope(payload *PayloadSchema) (*EnvelopeSchema, error) {
	// Create the envelope schema
	envelope := &AvroSchema{
		Type:      "record",
		Name:      payload.EventTypeName,
		Namespace: payload.Original.Namespace,
		Doc:       fmt.Sprintf("Event envelope for %s", payload.EventName),
		Fields: []AvroField{
			{
				Name: "metadata",
				Type: p.metadata.FullName(),
				Doc:  "Event metadata containing technical and observability fields",
			},
			{
				Name: "payload",
				Type: p.createInlinePayload(payload.Original),
				Doc:  fmt.Sprintf("Business data for %s event", toSpacedName(payload.EventName)),
			},
		},
	}

	// Generate envelope JSON
	envelopeJSON, err := envelope.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope schema: %w", err)
	}

	// Generate combined JSON (with metadata inlined)
	combinedJSON, err := p.createCombinedSchema(envelope, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined schema: %w", err)
	}

	// Validate the combined schema using hamba/avro
	if _, err := avro.Parse(string(combinedJSON)); err != nil {
		return nil, fmt.Errorf("invalid combined schema: %w", err)
	}

	return &EnvelopeSchema{
		Payload:      payload,
		Schema:       envelope,
		SchemaJSON:   envelopeJSON,
		CombinedJSON: combinedJSON,
	}, nil
}

// createInlinePayload creates an inline payload type definition for the envelope.
func (p *Parser) createInlinePayload(original *AvroSchema) map[string]any {
	// Create a copy without namespace (will be inlined)
	return map[string]any{
		"type":   "record",
		"name":   original.Name,
		"doc":    original.Doc,
		"fields": original.Fields,
	}
}

// createCombinedSchema creates a schema with EventMetadata fully inlined.
// This is needed for Avro serialization as the schema must be self-contained.
func (p *Parser) createCombinedSchema(envelope *AvroSchema, payload *PayloadSchema) ([]byte, error) {
	// Create metadata without namespace for inlining
	metadataInline := map[string]any{
		"type":   "record",
		"name":   p.metadata.Name,
		"doc":    p.metadata.Doc,
		"fields": p.metadata.Fields,
	}

	// Create payload inline
	payloadInline := map[string]any{
		"type":   "record",
		"name":   payload.Original.Name,
		"doc":    payload.Original.Doc,
		"fields": payload.Original.Fields,
	}

	// Create combined schema
	combined := map[string]any{
		"type":      "record",
		"name":      envelope.Name,
		"namespace": envelope.Namespace,
		"doc":       envelope.Doc,
		"fields": []map[string]any{
			{
				"name": "metadata",
				"type": metadataInline,
				"doc":  "Event metadata containing technical and observability fields",
			},
			{
				"name": "payload",
				"type": payloadInline,
				"doc":  envelope.Fields[1].Doc,
			},
		},
	}

	return marshalJSONIndent(combined)
}

// extractBaseName extracts the base name from a payload file path.
// e.g., "/path/to/product_created_payload.avsc" -> "product_created"
func extractBaseName(filePath string) string {
	base := filepath.Base(filePath)
	// Remove .avsc extension
	base = strings.TrimSuffix(base, ".avsc")
	// Remove _payload suffix
	base = strings.TrimSuffix(base, "_payload")
	return base
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

// toSpacedName converts PascalCase to spaced lowercase.
// e.g., "ProductCreated" -> "product created"
func toSpacedName(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
