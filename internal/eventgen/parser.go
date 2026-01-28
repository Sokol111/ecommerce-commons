package eventgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParsePayloads reads and parses all *_payload.avsc files.
func ParsePayloads(payloadsDir string) ([]*PayloadSchema, error) {
	pattern := filepath.Join(payloadsDir, "*_payload.avsc")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob payload files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *_payload.avsc files found in %s", payloadsDir)
	}

	payloads := make([]*PayloadSchema, 0, len(files))
	for _, file := range files {
		payload, err := parsePayload(file)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	return payloads, nil
}

func parsePayload(filePath string) (*PayloadSchema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	schema, err := ParseAvroSchema(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	return &PayloadSchema{
		Original: schema,
		FilePath: filePath,
		BaseName: extractBaseName(filePath),
		Topic:    schema.Topic,
	}, nil
}

func extractBaseName(filePath string) string {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, ".avsc")
	base = strings.TrimSuffix(base, "_payload")
	return base
}
