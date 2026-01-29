package eventgen

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed schemas/event_metadata.avsc
var embeddedMetadataSchema []byte

func WriteSchemaFiles(cfg *Config, payloads []*AvroSchema) error {
	fmt.Println("Writing schema files...")

	metadata, err := ParseAvroSchema(embeddedMetadataSchema)
	if err != nil {
		return fmt.Errorf("failed to parse metadata schema: %w", err)
	}

	for _, p := range payloads {
		data, err := CombineWithMetadata(metadata, p)
		if err != nil {
			return fmt.Errorf("failed to combine %s: %w", p.BaseName(), err)
		}

		schemaPath := filepath.Join(cfg.OutputDir, "schemas", p.BaseName()+".avsc")
		if err := os.WriteFile(schemaPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write schema: %w", err)
		}

		fmt.Printf("  Created %s.avsc\n", p.BaseName())
	}

	return nil
}
