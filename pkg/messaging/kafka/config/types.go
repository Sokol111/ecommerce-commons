package config

import "time"

// Config represents the main Kafka configuration.
type Config struct {
	Brokers         string               `koanf:"brokers"`          // Comma-separated list of Kafka broker addresses (e.g., "localhost:9092,localhost:9093")
	SchemaRegistry  SchemaRegistryConfig `koanf:"schema-registry"`  // Schema Registry configuration for Avro serialization/deserialization
	ConsumersConfig ConsumersConfig      `koanf:"consumers-config"` // Global and individual consumer configurations
	ProducerConfig  ProducerConfig       `koanf:"producer-config"`  // Producer-specific configuration
}

// ConsumersConfig holds global default settings and individual consumer configurations.
type ConsumersConfig struct {
	DefaultGroupID           string           `koanf:"default-group-id"`            // Default consumer group ID (applied to consumers without explicit group-id)
	DefaultAutoOffsetReset   string           `koanf:"default-auto-offset-reset"`   // Default offset reset policy: "earliest" or "latest"
	DefaultMaxRetries        *uint            `koanf:"default-max-retries"`         // Default maximum retries for message processing (0-99)
	DefaultInitialBackoff    time.Duration    `koanf:"default-initial-backoff"`     // Default initial backoff duration for retries (100ms-30s)
	DefaultMaxBackoff        time.Duration    `koanf:"default-max-backoff"`         // Default maximum backoff duration for retries (1s-5m)
	DefaultProcessingTimeout time.Duration    `koanf:"default-processing-timeout"`  // Default timeout for processing a single message (1s-10m)
	DefaultChannelBufferSize int              `koanf:"default-channel-buffer-size"` // Default internal message channel buffer size (10-10000)
	DefaultMaxPollRecords    int              `koanf:"default-max-poll-records"`    // Default max records per poll iteration (1-10000)
	ConsumerConfig           []ConsumerConfig `koanf:"consumers"`                   // Individual consumer configurations
}

// ConsumerConfig represents configuration for an individual Kafka consumer.
type ConsumerConfig struct {
	Name                    string        `koanf:"name"`                      // Unique consumer name/identifier (required)
	Topic                   string        `koanf:"topic"`                     // Kafka topic to consume from (required)
	GroupID                 string        `koanf:"group-id"`                  // Consumer group ID (defaults to DefaultGroupID)
	AutoOffsetReset         string        `koanf:"auto-offset-reset"`         // Offset reset policy: "earliest" or "latest" (defaults to DefaultAutoOffsetReset)
	EnableDLQ               bool          `koanf:"enable-dlq"`                // Enable Dead Letter Queue for failed messages
	DLQTopic                string        `koanf:"dlq-topic"`                 // DLQ topic name (defaults to "{topic}.dlq" if EnableDLQ is true)
	ReadinessTimeoutSeconds int           `koanf:"readiness-timeout-seconds"` // Timeout in seconds for waiting topic readiness (0 = no timeout, max 600s)
	FailOnTopicError        bool          `koanf:"fail-on-topic-error"`       // Whether to fail application startup if topic is not available
	MaxRetries              *uint         `koanf:"max-retries"`               // Maximum retries for message processing (0-99, defaults to DefaultMaxRetries)
	InitialBackoff          time.Duration `koanf:"initial-backoff"`           // Initial backoff duration between retries (100ms-30s, defaults to DefaultInitialBackoff)
	MaxBackoff              time.Duration `koanf:"max-backoff"`               // Maximum backoff duration between retries (1s-5m, defaults to DefaultMaxBackoff)
	ProcessingTimeout       time.Duration `koanf:"processing-timeout"`        // Timeout for processing a single message attempt (1s-10m, defaults to DefaultProcessingTimeout)
	ChannelBufferSize       int           `koanf:"channel-buffer-size"`       // Internal message channel buffer size (10-10000, defaults to DefaultChannelBufferSize)
	MaxPollRecords          int           `koanf:"max-poll-records"`          // Max records fetched per poll iteration (1-10000, defaults to DefaultMaxPollRecords)
}

// ProducerConfig represents configuration for Kafka producer.
type ProducerConfig struct {
	ReadinessTimeoutSeconds int           `koanf:"readiness-timeout-seconds"` // Timeout in seconds for waiting brokers readiness (0 = no timeout, max 600s, default 30s)
	FailOnBrokerError       bool          `koanf:"fail-on-broker-error"`      // Whether to fail application startup if brokers are not available (default false)
	Linger                  time.Duration `koanf:"linger"`                    // Time to wait for batching records before sending (0-1s, default 5ms)
	Compression             string        `koanf:"compression"`               // Compression codec: "none", "snappy", "lz4", "zstd" (default "snappy")
	DeliveryTimeout         time.Duration `koanf:"delivery-timeout"`          // Max time a record can sit in buffer before timing out (1s-5m, default 30s)
	MaxBufferedRecords      int           `koanf:"max-buffered-records"`      // Max records buffered in memory before blocking (100-1000000, default 10000)
}

// SchemaRegistryConfig represents Confluent Schema Registry configuration.
type SchemaRegistryConfig struct {
	URL                 string `koanf:"url"`                   // Schema Registry URL (required, e.g., "http://schema-registry:8081")
	CacheCapacity       int    `koanf:"cache-capacity"`        // Schema cache capacity (100-100000, default 1000)
	AutoRegisterSchemas bool   `koanf:"auto-register-schemas"` // Automatically register schemas on startup
}
