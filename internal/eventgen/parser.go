package eventgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hamba/avro/v2"
)

// ParsePayloads reads and parses all *_payload.avsc files, creating envelope schemas.
// Returns fully populated PayloadSchema with CombinedJSON ready for use.
func ParsePayloads(cfg *Config, metadata *AvroSchema) ([]*PayloadSchema, error) {
	pattern := filepath.Join(cfg.PayloadsDir, "*_payload.avsc")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob payload files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *_payload.avsc files found in %s", cfg.PayloadsDir)
	}

	payloads := make([]*PayloadSchema, 0, len(files))
	for _, file := range files {
		payload, err := parsePayload(file, metadata)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	return payloads, nil
}

// parsePayload parses a payload schema file and creates envelope with CombinedJSON.
func parsePayload(filePath string, metadata *AvroSchema) (*PayloadSchema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	schema, err := ParseAvroSchema(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	payload := &PayloadSchema{
		Original: schema,
		FilePath: filePath,
		BaseName: extractBaseName(filePath),
		Topic:    schema.Topic,
	}

	// Create combined schema with metadata and payload inlined
	combined := map[string]any{
		"type":      "record",
		"name":      payload.EventTypeName(),
		"namespace": schema.Namespace,
		"doc":       fmt.Sprintf("Event envelope for %s", payload.EventName()),
		"fields": []map[string]any{
			{
				"name": "metadata",
				"type": map[string]any{
					"type":   "record",
					"name":   metadata.Name,
					"doc":    metadata.Doc,
					"fields": metadata.Fields,
				},
				"doc": "Event metadata containing technical and observability fields",
			},
			{
				"name": "payload",
				"type": map[string]any{
					"type":   "record",
					"name":   schema.Name,
					"doc":    schema.Doc,
					"fields": schema.Fields,
				},
				"doc": fmt.Sprintf("Business data for %s event", toSpacedName(payload.EventName())),
			},
		},
	}

	combinedJSON, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema for %s: %w", filePath, err)
	}

	// Validate the combined schema
	if _, err := avro.Parse(string(combinedJSON)); err != nil {
		return nil, fmt.Errorf("invalid schema for %s: %w", filePath, err)
	}

	payload.CombinedJSON = combinedJSON
	return payload, nil
}

// extractBaseName extracts the base name from a payload file path.
// e.g., "/path/to/product_created_payload.avsc" -> "product_created"
func extractBaseName(filePath string) string {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, ".avsc")
	base = strings.TrimSuffix(base, "_payload")
	return base
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
