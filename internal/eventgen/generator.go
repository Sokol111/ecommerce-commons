package eventgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func Generate(cfg *Config) error {
	fmt.Println("Starting code generation...")

	fmt.Printf("Parsing payload schemas from %s\n", cfg.PayloadsDir)
	payloads, err := ParsePayloads(cfg.PayloadsDir)
	if err != nil {
		return fmt.Errorf("failed to parse payloads: %w", err)
	}
	fmt.Printf("Found %d payload schemas\n", len(payloads))

	if err := createOutputDir(cfg); err != nil {
		return err
	}

	// Sort payloads by name for consistent output
	sort.Slice(payloads, func(i, j int) bool {
		return payloads[i].EventName() < payloads[j].EventName()
	})

	if err := WriteSchemaFiles(cfg, payloads); err != nil {
		return err
	}

	// Generate Go code using jennifer
	if err := GenerateGoTypes(cfg, payloads); err != nil {
		return err
	}

	// Generate constants.gen.go
	if err := GenerateConstants(cfg, payloads); err != nil {
		return err
	}

	// Generate schemas.gen.go
	if err := GenerateSchemaEmbeddings(cfg, payloads); err != nil {
		return err
	}

	fmt.Println("✓ Code generation complete!")
	return nil
}

func createOutputDir(cfg *Config) error {
	schemasDir := filepath.Join(cfg.OutputDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", schemasDir, err)
	}
	return nil
}
