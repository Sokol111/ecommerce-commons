package eventgen

import (
	"fmt"
	"path/filepath"
)

// Config holds the configuration for the event generator.
type Config struct {
	// PayloadsDir is the directory containing *_payload.avsc files.
	PayloadsDir string
	// OutputDir is the directory where generated code will be written.
	OutputDir string
	// Package is the Go package name for generated code. Defaults to "events".
	Package string
}

// Validate checks that the configuration is valid for generation.
func (c *Config) Validate() error {
	if c.PayloadsDir == "" {
		return fmt.Errorf("payloads directory is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if c.Package == "" {
		c.Package = "events"
	}

	// Convert to absolute paths
	var err error
	if c.PayloadsDir, err = filepath.Abs(c.PayloadsDir); err != nil {
		return fmt.Errorf("failed to resolve payloads directory: %w", err)
	}
	if c.OutputDir, err = filepath.Abs(c.OutputDir); err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}

	return nil
}
