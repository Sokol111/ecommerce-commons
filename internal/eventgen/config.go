// Package eventgen provides functionality to generate Go code from Avro payload schemas.
//
// The package reads Avro payload schemas (*_payload.avsc), combines them with
// a standard EventMetadata schema to create envelope schemas, and generates
// Go types, constants, and schema embeddings.
//
// Basic usage:
//
//	cfg := &eventgen.Config{
//		PayloadsDir: "./avro",
//		OutputDir:   "./gen/events",
//		Package:     "events",
//	}
//
//	gen, err := eventgen.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if err := gen.Generate(); err != nil {
//		log.Fatal(err)
//	}
package eventgen

import (
	"fmt"
	"path/filepath"
)

// Config holds the configuration for the event generator.
type Config struct {
	// PayloadsDir is the directory containing *_payload.avsc files.
	// This is required.
	PayloadsDir string

	// OutputDir is the directory where generated code will be written.
	// This is required for generation.
	OutputDir string

	// Package is the Go package name for generated code.
	// Defaults to "events" if not specified.
	Package string

	// AsyncAPIFile is the optional path to an AsyncAPI spec file.
	// Used to extract topic names for generated constants.
	AsyncAPIFile string

	// MetadataFile is the optional path to a custom EventMetadata schema.
	// If not specified, the embedded default schema is used.
	MetadataFile string

	// MetadataNamespace is the namespace for EventMetadata.
	// Defaults to "com.ecommerce.events".
	MetadataNamespace string

	// Verbose enables detailed logging during generation.
	Verbose bool
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.PayloadsDir == "" {
		return fmt.Errorf("payloads directory is required")
	}

	if c.Package == "" {
		c.Package = "events"
	}

	if c.MetadataNamespace == "" {
		c.MetadataNamespace = "com.ecommerce.events"
	}

	return nil
}

// ValidateForGeneration checks that the configuration is valid for code generation.
func (c *Config) ValidateForGeneration() error {
	if err := c.Validate(); err != nil {
		return err
	}

	if c.OutputDir == "" {
		return fmt.Errorf("output directory is required for generation")
	}

	return nil
}

// AbsolutePaths converts relative paths to absolute paths.
func (c *Config) AbsolutePaths() error {
	var err error

	if c.PayloadsDir != "" {
		c.PayloadsDir, err = filepath.Abs(c.PayloadsDir)
		if err != nil {
			return fmt.Errorf("failed to resolve payloads directory: %w", err)
		}
	}

	if c.OutputDir != "" {
		c.OutputDir, err = filepath.Abs(c.OutputDir)
		if err != nil {
			return fmt.Errorf("failed to resolve output directory: %w", err)
		}
	}

	if c.AsyncAPIFile != "" {
		c.AsyncAPIFile, err = filepath.Abs(c.AsyncAPIFile)
		if err != nil {
			return fmt.Errorf("failed to resolve asyncapi file: %w", err)
		}
	}

	if c.MetadataFile != "" {
		c.MetadataFile, err = filepath.Abs(c.MetadataFile)
		if err != nil {
			return fmt.Errorf("failed to resolve metadata file: %w", err)
		}
	}

	return nil
}
