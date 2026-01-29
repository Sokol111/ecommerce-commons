package eventgen

import (
	"fmt"
	"os"
	"path/filepath"
)

// ParsePayloads reads and parses all *_payload.avsc files.
func ParsePayloads(payloadsDir string) ([]*AvroSchema, error) {
	pattern := filepath.Join(payloadsDir, "*_payload.avsc")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob payload files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *_payload.avsc files found in %s", payloadsDir)
	}

	payloads := make([]*AvroSchema, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		schema, err := ParseAvroSchema(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		payloads = append(payloads, schema)
	}

	return payloads, nil
}
