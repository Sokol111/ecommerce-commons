package config

import "time"

// Config represents the main Kafka configuration.
type Config struct {
	Brokers         string               `mapstructure:"brokers"`          // Comma-separated list of Kafka broker addresses (e.g., "localhost:9092,localhost:9093")
	SchemaRegistry  SchemaRegistryConfig `mapstructure:"schema-registry"`  // Schema Registry configuration for Avro serialization/deserialization
	ConsumersConfig ConsumersConfig      `mapstructure:"consumers-config"` // Global and individual consumer configurations
	ProducerConfig  ProducerConfig       `mapstructure:"producer-config"`  // Producer-specific configuration
}

// ConsumersConfig holds global default settings and individual consumer configurations.
type ConsumersConfig struct {
	DefaultGroupID           string           `mapstructure:"default-group-id"`            // Default consumer group ID (applied to consumers without explicit group-id)
	DefaultAutoOffsetReset   string           `mapstructure:"default-auto-offset-reset"`   // Default offset reset policy: "earliest" or "latest"
	DefaultMaxRetryAttempts  int              `mapstructure:"default-max-retry-attempts"`  // Default maximum retry attempts for message processing (1-100)
	DefaultInitialBackoff    time.Duration    `mapstructure:"default-initial-backoff"`     // Default initial backoff duration for retries (100ms-30s)
	DefaultMaxBackoff        time.Duration    `mapstructure:"default-max-backoff"`         // Default maximum backoff duration for retries (1s-5m)
	DefaultProcessingTimeout time.Duration    `mapstructure:"default-processing-timeout"`  // Default timeout for processing a single message (1s-10m)
	DefaultChannelBufferSize int              `mapstructure:"default-channel-buffer-size"` // Default internal message channel buffer size (10-10000)
	ConsumerConfig           []ConsumerConfig `mapstructure:"consumers"`                   // Individual consumer configurations
}

// ConsumerConfig represents configuration for an individual Kafka consumer.
type ConsumerConfig struct {
	Name                    string        `mapstructure:"name"`                      // Unique consumer name/identifier (required)
	Topic                   string        `mapstructure:"topic"`                     // Kafka topic to consume from (required)
	GroupID                 string        `mapstructure:"group-id"`                  // Consumer group ID (defaults to DefaultGroupID)
	AutoOffsetReset         string        `mapstructure:"auto-offset-reset"`         // Offset reset policy: "earliest" or "latest" (defaults to DefaultAutoOffsetReset)
	EnableDLQ               bool          `mapstructure:"enable-dlq"`                // Enable Dead Letter Queue for failed messages
	DLQTopic                string        `mapstructure:"dlq-topic"`                 // DLQ topic name (defaults to "{topic}.dlq" if EnableDLQ is true)
	ReadinessTimeoutSeconds int           `mapstructure:"readiness-timeout-seconds"` // Timeout in seconds for waiting topic readiness (0 = no timeout, max 600s)
	FailOnTopicError        bool          `mapstructure:"fail-on-topic-error"`       // Whether to fail application startup if topic is not available
	MaxRetryAttempts        int           `mapstructure:"max-retry-attempts"`        // Maximum retry attempts for message processing (1-100, defaults to DefaultMaxRetryAttempts)
	InitialBackoff          time.Duration `mapstructure:"initial-backoff"`           // Initial backoff duration between retries (100ms-30s, defaults to DefaultInitialBackoff)
	MaxBackoff              time.Duration `mapstructure:"max-backoff"`               // Maximum backoff duration between retries (1s-5m, defaults to DefaultMaxBackoff)
	ProcessingTimeout       time.Duration `mapstructure:"processing-timeout"`        // Timeout for processing a single message attempt (1s-10m, defaults to DefaultProcessingTimeout)
	ChannelBufferSize       int           `mapstructure:"channel-buffer-size"`       // Internal message channel buffer size (10-10000, defaults to DefaultChannelBufferSize)
}

// ProducerConfig represents configuration for Kafka producer.
type ProducerConfig struct {
	ReadinessTimeoutSeconds int  `mapstructure:"readiness-timeout-seconds"` // Timeout in seconds for waiting brokers readiness (0 = no timeout, max 600s, default 30s)
	FailOnBrokerError       bool `mapstructure:"fail-on-broker-error"`      // Whether to fail application startup if brokers are not available (default false)
}

// SchemaRegistryConfig represents Confluent Schema Registry configuration.
type SchemaRegistryConfig struct {
	URL                 string `mapstructure:"url"`                   // Schema Registry URL (required, e.g., "http://schema-registry:8081")
	CacheCapacity       int    `mapstructure:"cache-capacity"`        // Schema cache capacity (100-100000, default 1000)
	AutoRegisterSchemas bool   `mapstructure:"auto_register_schemas"` // Automatically register schemas on startup
}
