package eventgen

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed schemas/event_metadata.avsc
var embeddedMetadataSchema []byte

const (
	commonsEventsImport  = "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	commonsMappingImport = "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
)

// Generator orchestrates the code generation process.
type Generator struct {
	config    *Config
	metadata  *AvroSchema
	parser    *Parser
	asyncAPI  *AsyncAPIParser
	templates *TemplateRenderer
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
	asyncAPI := NewAsyncAPIParser(cfg)
	templates := NewTemplateRenderer(cfg)

	return &Generator{
		config:    cfg,
		metadata:  metadata,
		parser:    parser,
		asyncAPI:  asyncAPI,
		templates: templates,
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

	// Parse AsyncAPI spec (if provided)
	var topics []*TopicInfo
	var asyncSpec *AsyncAPISpec
	var eventTopicMap map[string]string
	if g.config.AsyncAPIFile != "" {
		g.log("Parsing AsyncAPI spec from %s", g.config.AsyncAPIFile)
		asyncSpec, err = g.asyncAPI.Parse()
		if err != nil {
			return fmt.Errorf("failed to parse AsyncAPI: %w", err)
		}
		topics = g.asyncAPI.ExtractTopics(asyncSpec)
		eventTopicMap = g.asyncAPI.BuildEventTopicMapping(asyncSpec)
		g.log("Found %d topics", len(topics))
	}

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

	// Generate Go code using avrogen
	if err := g.generateGoTypes(envelopes); err != nil {
		return err
	}

	// Generate constants.gen.go
	if err := g.generateConstants(envelopes, topics, asyncSpec, eventTopicMap); err != nil {
		return err
	}

	// Generate schemas.gen.go
	if err := g.generateSchemaEmbeddings(envelopes); err != nil {
		return err
	}

	// Copy AsyncAPI spec if provided
	if g.config.AsyncAPIFile != "" {
		if err := g.copyAsyncAPISpec(); err != nil {
			return err
		}
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

// generateGoTypes uses avrogen to generate Go types from combined schemas.
func (g *Generator) generateGoTypes(envelopes []*EnvelopeSchema) error {
	g.log("Generating Go types with avrogen...")

	// Build list of combined schema files
	schemaFiles := make([]string, 0, len(envelopes))
	for _, env := range envelopes {
		schemaFiles = append(schemaFiles,
			filepath.Join(g.config.OutputDir, "schemas", env.Payload.BaseName+".avsc"))
	}

	// Run avrogen
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

	// Post-process to use commons EventMetadata
	if err := g.postProcessTypes(outputFile, envelopes); err != nil {
		return fmt.Errorf("failed to post-process types: %w", err)
	}

	g.log("  Created types.gen.go")
	return nil
}

// postProcessTypes modifies the generated types.gen.go to use commons EventMetadata.
func (g *Generator) postProcessTypes(filePath string, envelopes []*EnvelopeSchema) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	result := string(content)

	// Remove the generated EventMetadata struct
	// Match from "// Common event metadata..." to the closing brace of the struct
	metadataRegex := regexp.MustCompile(`(?s)// Common event metadata[^}]+}\n\n`)
	result = metadataRegex.ReplaceAllString(result, "")

	// Add import for commons events package
	// Find the import block and add our import
	if strings.Contains(result, "import (") {
		result = strings.Replace(result,
			"import (",
			fmt.Sprintf("import (\n\t\"%s\"", commonsEventsImport),
			1)
	} else if strings.Contains(result, "import \"") {
		// Single import - convert to block
		singleImportRegex := regexp.MustCompile(`import "([^"]+)"`)
		result = singleImportRegex.ReplaceAllString(result,
			fmt.Sprintf("import (\n\t\"%s\"\n\t\"$1\"\n)", commonsEventsImport))
	} else {
		// No imports - add import block after package
		result = strings.Replace(result,
			fmt.Sprintf("package %s\n", g.config.Package),
			fmt.Sprintf("package %s\n\nimport (\n\t\"%s\"\n)\n", g.config.Package, commonsEventsImport),
			1)
	}

	// Replace EventMetadata with events.EventMetadata in struct fields
	result = strings.ReplaceAll(result, "Metadata EventMetadata", "Metadata events.EventMetadata")

	// Remove unused "time" import if no time.Time is used in the code
	// (EventMetadata uses time.Time but we're replacing it with commons)
	if !strings.Contains(result, "time.Time") {
		// Remove from import block
		result = regexp.MustCompile(`\t"time"\n`).ReplaceAllString(result, "")
		// Also handle single import case
		result = regexp.MustCompile(`import "time"\n`).ReplaceAllString(result, "")
	}

	// Add Event interface implementation for each event type
	var implementations strings.Builder
	implementations.WriteString("\n// Event interface implementations\n")
	for _, env := range envelopes {
		implementations.WriteString(fmt.Sprintf(`
func (e *%s) GetMetadata() *events.EventMetadata { return &e.Metadata }
`, env.Payload.EventTypeName))
	}

	result += implementations.String()

	return os.WriteFile(filePath, []byte(result), 0644)
}

// generateConstants generates the constants.gen.go file.
func (g *Generator) generateConstants(envelopes []*EnvelopeSchema, topics []*TopicInfo, asyncSpec *AsyncAPISpec, eventTopicMap map[string]string) error {
	g.log("Generating constants...")

	outputFile := filepath.Join(g.config.OutputDir, "constants.gen.go")

	content, err := g.templates.RenderConstants(envelopes, topics, asyncSpec, eventTopicMap)
	if err != nil {
		return fmt.Errorf("failed to render constants: %w", err)
	}

	if err := os.WriteFile(outputFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write constants: %w", err)
	}

	g.log("  Created constants.gen.go")
	return nil
}

// generateSchemaEmbeddings generates the schemas.gen.go file.
func (g *Generator) generateSchemaEmbeddings(envelopes []*EnvelopeSchema) error {
	g.log("Generating schema embeddings...")

	outputFile := filepath.Join(g.config.OutputDir, "schemas.gen.go")

	content, err := g.templates.RenderSchemas(envelopes, g.config.AsyncAPIFile != "")
	if err != nil {
		return fmt.Errorf("failed to render schemas: %w", err)
	}

	if err := os.WriteFile(outputFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write schemas: %w", err)
	}

	g.log("  Created schemas.gen.go")
	return nil
}

// copyAsyncAPISpec copies the AsyncAPI spec to the output directory.
func (g *Generator) copyAsyncAPISpec() error {
	data, err := os.ReadFile(g.config.AsyncAPIFile)
	if err != nil {
		return fmt.Errorf("failed to read AsyncAPI spec: %w", err)
	}

	destPath := filepath.Join(g.config.OutputDir, "asyncapi.yaml")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write AsyncAPI spec: %w", err)
	}

	g.log("  Copied asyncapi.yaml")
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
