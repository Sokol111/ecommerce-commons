// Package main provides the eventgen CLI tool for generating Go code from Avro payload schemas.
//
// Usage:
//
//	eventgen generate --payloads ./avro --output ./gen/events --package events
//
// The tool reads *_payload.avsc files and generates complete event envelope schemas
// with embedded EventMetadata, along with Go types, constants, and schema embeddings.
package main

import (
	"fmt"
	"os"

	"github.com/Sokol111/ecommerce-commons/internal/eventgen"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "eventgen",
		Short:   "Generate Go code from Avro payload schemas",
		Long:    `eventgen generates Go types, constants, and schema embeddings from Avro payload schemas.`,
		Version: version,
	}

	rootCmd.AddCommand(newGenerateCmd())

	return rootCmd
}

func newGenerateCmd() *cobra.Command {
	cfg := &eventgen.Config{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Go code from Avro payload schemas",
		Long: `Generate Go code from Avro payload schemas.

This command reads *_payload.avsc files from the payloads directory,
combines them with EventMetadata to create envelope schemas,
and generates Go types, constants, and schema embeddings.

Example:
  eventgen generate --payloads ./avro --output ./gen/events --package events`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cfg)
		},
	}

	// Required flags
	cmd.Flags().StringVarP(&cfg.PayloadsDir, "payloads", "p", "", "Directory containing *_payload.avsc files (required)")
	cmd.Flags().StringVarP(&cfg.OutputDir, "output", "o", "", "Output directory for generated code (required)")

	// Optional flags
	cmd.Flags().StringVarP(&cfg.Package, "package", "n", "events", "Go package name for generated code")
	cmd.Flags().StringVarP(&cfg.MetadataFile, "metadata", "m", "", "Custom EventMetadata schema file (uses embedded if not specified)")
	cmd.Flags().StringVar(&cfg.MetadataNamespace, "metadata-namespace", "com.ecommerce.events", "Namespace for EventMetadata")
	cmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output")

	_ = cmd.MarkFlagRequired("payloads")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func runGenerate(cfg *eventgen.Config) error {
	gen, err := eventgen.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	if err := gen.Generate(); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	return nil
}
