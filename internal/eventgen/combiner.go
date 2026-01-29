package eventgen

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/hamba/avro/v2"
)

// Combine creates an envelope schema JSON by combining metadata with payload.
func CombineWithMetadata(metadata *AvroSchema, payload *AvroSchema) ([]byte, error) {
	combined := map[string]any{
		"type":      "record",
		"name":      payload.EventTypeName(),
		"namespace": payload.Namespace,
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
					"name":   payload.Name,
					"doc":    payload.Doc,
					"fields": payload.Fields,
				},
				"doc": fmt.Sprintf("Business data for %s event", payload.EventName()),
			},
		},
	}

	data, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope schema: %w", err)
	}

	// Validate the combined schema
	if _, err := avro.Parse(string(data)); err != nil {
		return nil, fmt.Errorf("invalid envelope schema: %w", err)
	}

	return data, nil
}
