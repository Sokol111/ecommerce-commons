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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/dave/jennifer/jen"
)

//go:embed schemas/event_metadata.avsc
var embeddedMetadataSchema []byte

const (
	commonsEventsImport  = "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	commonsMappingImport = "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
)

// Config holds the configuration for the event generator.
type Config struct {
	// PayloadsDir is the directory containing *_payload.avsc files.
	PayloadsDir string
	// OutputDir is the directory where generated code will be written.
	OutputDir string
	// Package is the Go package name for generated code. Defaults to "events".
	Package string
	// MetadataFile is the optional path to a custom EventMetadata schema.
	MetadataFile string
	// MetadataNamespace is the namespace for EventMetadata. Defaults to "com.ecommerce.events".
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
		if c.PayloadsDir, err = filepath.Abs(c.PayloadsDir); err != nil {
			return fmt.Errorf("failed to resolve payloads directory: %w", err)
		}
	}
	if c.OutputDir != "" {
		if c.OutputDir, err = filepath.Abs(c.OutputDir); err != nil {
			return fmt.Errorf("failed to resolve output directory: %w", err)
		}
	}
	if c.MetadataFile != "" {
		if c.MetadataFile, err = filepath.Abs(c.MetadataFile); err != nil {
			return fmt.Errorf("failed to resolve metadata file: %w", err)
		}
	}
	return nil
}

// Generator orchestrates the code generation process.
type Generator struct {
	config   *Config
	metadata *AvroSchema
	parser   *Parser
}

// New creates a new Generator with the given configuration.
func New(cfg *Config) (*Generator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if err := cfg.AbsolutePaths(); err != nil {
		return nil, err
	}

	// Load metadata schema
	metadata, err := loadMetadataSchema(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata schema: %w", err)
	}

	parser := NewParser(cfg, metadata)

	return &Generator{
		config:   cfg,
		metadata: metadata,
		parser:   parser,
	}, nil
}

// Generate runs the complete code generation process.
func (g *Generator) Generate() error {
	if err := g.config.ValidateForGeneration(); err != nil {
		return err
	}

	g.log("Starting code generation...")

	// Parse payload schemas
	g.log("Parsing payload schemas from %s", g.config.PayloadsDir)
	payloads, err := g.parser.ParsePayloads()
	if err != nil {
		return fmt.Errorf("failed to parse payloads: %w", err)
	}
	g.log("Found %d payload schemas", len(payloads))

	// Create output directories
	if err := g.createOutputDirs(); err != nil {
		return err
	}

	// Generate envelope schemas
	g.log("Generating envelope schemas...")
	envelopes := make([]*EnvelopeSchema, 0, len(payloads))
	for _, payload := range payloads {
		envelope, err := g.parser.CreateEnvelope(payload)
		if err != nil {
			return fmt.Errorf("failed to create envelope for %s: %w", payload.BaseName, err)
		}
		envelopes = append(envelopes, envelope)
	}

	// Sort envelopes by name for consistent output
	sort.Slice(envelopes, func(i, j int) bool {
		return envelopes[i].Payload.EventName < envelopes[j].Payload.EventName
	})

	// Write schema files
	if err := g.writeSchemaFiles(envelopes); err != nil {
		return err
	}

	// Generate Go code using jennifer
	if err := g.generateGoTypes(envelopes); err != nil {
		return err
	}

	// Generate constants.gen.go
	if err := g.generateConstants(envelopes); err != nil {
		return err
	}

	// Generate schemas.gen.go
	if err := g.generateSchemaEmbeddings(envelopes); err != nil {
		return err
	}

	g.log("✓ Code generation complete!")
	return nil
}

// Validate validates payload schemas without generating code.
func (g *Generator) Validate() error {
	payloads, err := g.parser.ParsePayloads()
	if err != nil {
		return err
	}

	for _, payload := range payloads {
		// Try to create envelope to validate
		_, err := g.parser.CreateEnvelope(payload)
		if err != nil {
			return fmt.Errorf("invalid schema %s: %w", payload.FilePath, err)
		}
		g.log("✓ %s", filepath.Base(payload.FilePath))
	}

	return nil
}

// createOutputDirs creates the output directory structure.
func (g *Generator) createOutputDirs() error {
	dirs := []string{
		g.config.OutputDir,
		filepath.Join(g.config.OutputDir, "schemas"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// writeSchemaFiles writes combined schema files.
func (g *Generator) writeSchemaFiles(envelopes []*EnvelopeSchema) error {
	g.log("Writing schema files...")

	// Write combined schemas (with metadata inlined) - this is the only schema needed for Avro
	for _, env := range envelopes {
		schemaPath := filepath.Join(g.config.OutputDir, "schemas", env.Payload.BaseName+".avsc")
		if err := os.WriteFile(schemaPath, env.CombinedJSON, 0644); err != nil {
			return fmt.Errorf("failed to write schema: %w", err)
		}

		g.log("  Created %s.avsc", env.Payload.BaseName)
	}

	return nil
}

// generateGoTypes uses avrogen for payload types and jennifer for event wrappers.
func (g *Generator) generateGoTypes(envelopes []*EnvelopeSchema) error {
	g.log("Generating Go types...")

	// Step 1: Generate payload types with avrogen (from payload schemas directly)
	if err := g.generatePayloadTypes(envelopes); err != nil {
		return err
	}

	// Step 2: Generate event wrappers with jennifer
	if err := g.generateEventWrappers(envelopes); err != nil {
		return err
	}

	g.log("  Created types.gen.go and events.gen.go")
	return nil
}

// generatePayloadTypes uses avrogen to generate payload structs from payload schemas.
func (g *Generator) generatePayloadTypes(envelopes []*EnvelopeSchema) error {
	// Build list of payload schema files (original payloads, not envelopes)
	schemaFiles := make([]string, 0, len(envelopes))
	for _, env := range envelopes {
		schemaFiles = append(schemaFiles, env.Payload.FilePath)
	}

	outputFile := filepath.Join(g.config.OutputDir, "types.gen.go")
	args := []string{
		"-pkg", g.config.Package,
		"-o", outputFile,
		"-tags", "json:snake",
	}
	args = append(args, schemaFiles...)

	cmd := exec.Command("avrogen", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("avrogen failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// generateEventWrappers generates event envelope structs using jennifer.
func (g *Generator) generateEventWrappers(envelopes []*EnvelopeSchema) error {
	f := jen.NewFile(g.config.Package)
	f.HeaderComment("Code generated by eventgen. DO NOT EDIT.")

	for _, env := range envelopes {
		// Event struct with commons.EventMetadata
		f.Commentf("%s is the event envelope for %s.", env.Payload.EventTypeName, env.Payload.EventName)
		f.Type().Id(env.Payload.EventTypeName).Struct(
			jen.Id("Metadata").Qual(commonsEventsImport, "EventMetadata").Tag(map[string]string{
				"avro": "metadata",
				"json": "metadata",
			}),
			jen.Id("Payload").Id(env.Payload.PayloadTypeName).Tag(map[string]string{
				"avro": "payload",
				"json": "payload",
			}),
		)
		f.Line()

		// GetMetadata method for Event interface
		f.Func().
			Params(jen.Id("e").Op("*").Id(env.Payload.EventTypeName)).
			Id("GetMetadata").
			Params().
			Op("*").Qual(commonsEventsImport, "EventMetadata").
			Block(jen.Return(jen.Op("&").Id("e").Dot("Metadata")))
		f.Line()
	}

	outputFile := filepath.Join(g.config.OutputDir, "events.gen.go")
	return f.Save(outputFile)
}

// generateConstants generates the constants.gen.go file using jennifer.
func (g *Generator) generateConstants(envelopes []*EnvelopeSchema) error {
	g.log("Generating constants...")

	f := jen.NewFile(g.config.Package)
	f.HeaderComment("Code generated by eventgen. DO NOT EDIT.")

	// Import for reflect and mapping
	f.ImportName("reflect", "reflect")
	f.ImportName(commonsMappingImport, "mapping")

	// Event type constants
	f.Comment("Event type constants - match Avro schema names")
	f.Const().DefsFunc(func(group *jen.Group) {
		for _, env := range envelopes {
			group.Id("EventType" + env.Payload.EventName).Op("=").Lit(env.Payload.EventTypeName)
		}
	})
	f.Line()

	// Collect unique topics
	topicSet := make(map[string]bool)
	for _, env := range envelopes {
		if env.Payload.Topic != "" {
			topicSet[env.Payload.Topic] = true
		}
	}

	// Topic constants (if topics exist)
	if len(topicSet) > 0 {
		// Sort topics for consistent output
		topics := make([]string, 0, len(topicSet))
		for topic := range topicSet {
			topics = append(topics, topic)
		}
		sort.Strings(topics)

		f.Comment("Topic constants - Kafka topics from Avro schemas")
		f.Const().DefsFunc(func(group *jen.Group) {
			for _, topic := range topics {
				group.Id(TopicToConstName(topic)).Op("=").Lit(topic)
			}
		})
		f.Line()
	}

	// Schema name constants
	f.Comment("Schema name constants - Avro schema full names (namespace.name)")
	f.Const().DefsFunc(func(group *jen.Group) {
		for _, env := range envelopes {
			group.Id("SchemaName" + env.Payload.EventName).Op("=").Lit(env.Schema.FullName())
		}
	})
	f.Line()

	// SchemaBindings slice
	f.Comment("SchemaBindings contains all event schema bindings for registration with TypeMapping.")
	f.Comment("Use this to register all schemas in your microservice:")
	f.Comment("")
	f.Comment("\ttypeMapping.RegisterBindings(events.SchemaBindings)")
	f.Var().Id("SchemaBindings").Op("=").Index().Qual(commonsMappingImport, "SchemaBinding").ValuesFunc(func(group *jen.Group) {
		for _, env := range envelopes {
			var topicStmt *jen.Statement
			if env.Payload.Topic != "" {
				topicStmt = jen.Id(TopicToConstName(env.Payload.Topic))
			} else {
				topicStmt = jen.Lit("")
			}
			group.Values(jen.Dict{
				jen.Id("GoType"):     jen.Qual("reflect", "TypeOf").Call(jen.Id(env.Payload.EventTypeName).Values()),
				jen.Id("SchemaJSON"): jen.Id(env.Payload.EventName + "Schema"),
				jen.Id("SchemaName"): jen.Id("SchemaName" + env.Payload.EventName),
				jen.Id("Topic"):      topicStmt,
			})
		}
	})

	outputFile := filepath.Join(g.config.OutputDir, "constants.gen.go")
	if err := f.Save(outputFile); err != nil {
		return fmt.Errorf("failed to write constants: %w", err)
	}

	g.log("  Created constants.gen.go")
	return nil
}

// generateSchemaEmbeddings generates the schemas.gen.go file using jennifer.
func (g *Generator) generateSchemaEmbeddings(envelopes []*EnvelopeSchema) error {
	g.log("Generating schema embeddings...")

	f := jen.NewFile(g.config.Package)
	f.HeaderComment("Code generated by eventgen. DO NOT EDIT.")

	// Import embed
	f.ImportName("embed", "_")
	f.Anon("embed")

	// Event schema embeddings
	f.Comment("Event schemas with EventMetadata inlined (ready for Avro serialization)")
	for _, env := range envelopes {
		f.Comment(fmt.Sprintf("//go:embed schemas/%s.avsc", env.Payload.BaseName))
		f.Var().Id(env.Payload.EventName + "Schema").Index().Byte()
		f.Line()
	}

	outputFile := filepath.Join(g.config.OutputDir, "schemas.gen.go")
	if err := f.Save(outputFile); err != nil {
		return fmt.Errorf("failed to write schemas: %w", err)
	}

	g.log("  Created schemas.gen.go")
	return nil
}

// loadMetadataSchema loads the EventMetadata schema from file or embedded.
func loadMetadataSchema(cfg *Config) (*AvroSchema, error) {
	var data []byte
	var err error

	if cfg.MetadataFile != "" {
		data, err = os.ReadFile(cfg.MetadataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata file: %w", err)
		}
	} else {
		data = embeddedMetadataSchema
	}

	schema, err := ParseAvroSchema(data)
	if err != nil {
		return nil, err
	}

	// Override namespace if specified
	if cfg.MetadataNamespace != "" && schema.Namespace == "" {
		schema.Namespace = cfg.MetadataNamespace
	}

	return schema, nil
}

// log prints a message if verbose mode is enabled.
func (g *Generator) log(format string, args ...any) {
	if g.config.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}

// marshalJSONIndent is a helper to marshal JSON with indentation.
func marshalJSONIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// TopicToConstName converts a topic name to a Go constant name.
// e.g., "catalog.product.events" -> "TopicCatalogProductEvents"
func TopicToConstName(topic string) string {
	return "Topic" + toPascalCase(replaceDotsWithUnderscores(topic))
}

// replaceDotsWithUnderscores replaces dots with underscores for processing.
func replaceDotsWithUnderscores(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			result[i] = '_'
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}
