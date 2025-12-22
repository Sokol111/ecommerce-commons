package eventgen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AsyncAPISpec represents a parsed AsyncAPI specification.
type AsyncAPISpec struct {
	AsyncAPI string                     `yaml:"asyncapi"`
	Info     AsyncAPIInfo               `yaml:"info"`
	Channels map[string]AsyncAPIChannel `yaml:"channels"`
}

// AsyncAPIInfo contains API metadata.
type AsyncAPIInfo struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// AsyncAPIChannel represents a channel (topic) in AsyncAPI.
type AsyncAPIChannel struct {
	Description string             `yaml:"description"`
	Publish     *AsyncAPIOperation `yaml:"publish"`
	Subscribe   *AsyncAPIOperation `yaml:"subscribe"`
}

// AsyncAPIOperation represents a publish or subscribe operation.
type AsyncAPIOperation struct {
	OperationID string           `yaml:"operationId"`
	Summary     string           `yaml:"summary"`
	Message     *AsyncAPIMessage `yaml:"message"`
}

// AsyncAPIMessage represents a message in AsyncAPI.
type AsyncAPIMessage struct {
	Name        string `yaml:"name"`
	ContentType string `yaml:"contentType"`
	SchemaRef   string `yaml:"$ref"`
}

// AsyncAPIParser handles parsing of AsyncAPI specifications.
type AsyncAPIParser struct {
	config *Config
}

// NewAsyncAPIParser creates a new AsyncAPI parser.
func NewAsyncAPIParser(cfg *Config) *AsyncAPIParser {
	return &AsyncAPIParser{config: cfg}
}

// Parse reads and parses an AsyncAPI specification file.
func (p *AsyncAPIParser) Parse() (*AsyncAPISpec, error) {
	if p.config.AsyncAPIFile == "" {
		return nil, nil // No AsyncAPI file specified
	}

	data, err := os.ReadFile(p.config.AsyncAPIFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read AsyncAPI file: %w", err)
	}

	var spec AsyncAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse AsyncAPI spec: %w", err)
	}

	return &spec, nil
}

// ExtractTopics extracts topic information from an AsyncAPI spec.
func (p *AsyncAPIParser) ExtractTopics(spec *AsyncAPISpec) []*TopicInfo {
	if spec == nil {
		return nil
	}

	topics := make([]*TopicInfo, 0, len(spec.Channels))
	for name, channel := range spec.Channels {
		topic := &TopicInfo{
			Name:        name,
			Description: channel.Description,
		}

		// Extract event types from publish/subscribe operations
		if channel.Publish != nil && channel.Publish.Message != nil {
			topic.EventTypes = append(topic.EventTypes, channel.Publish.Message.Name)
		}
		if channel.Subscribe != nil && channel.Subscribe.Message != nil {
			topic.EventTypes = append(topic.EventTypes, channel.Subscribe.Message.Name)
		}

		topics = append(topics, topic)
	}

	return topics
}

// GetDefaultTopic returns the first topic from the spec, or empty string if none.
func (p *AsyncAPIParser) GetDefaultTopic(spec *AsyncAPISpec) string {
	if spec == nil || len(spec.Channels) == 0 {
		return ""
	}

	// Return the first channel name (topics are unordered in YAML)
	for name := range spec.Channels {
		return name
	}

	return ""
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
