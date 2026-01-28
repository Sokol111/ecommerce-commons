package eventgen

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/hamba/avro/v2"
)

//go:embed schemas/event_metadata.avsc
var embeddedMetadataSchema []byte

// Combiner creates envelope schemas by combining metadata with payload schemas.
type Combiner struct {
	metadata *AvroSchema
}

// NewCombiner creates a new Combiner with the embedded metadata schema.
func NewCombiner() (*Combiner, error) {
	metadata, err := ParseAvroSchema(embeddedMetadataSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata schema: %w", err)
	}
	return &Combiner{metadata: metadata}, nil
}

// Combine creates an envelope schema JSON by combining metadata with payload.
func (c *Combiner) Combine(payload *PayloadSchema) ([]byte, error) {
	schema := payload.Original

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
					"name":   c.metadata.Name,
					"doc":    c.metadata.Doc,
					"fields": c.metadata.Fields,
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
