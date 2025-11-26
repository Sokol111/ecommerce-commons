package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// Default values
	defaultSchemaRegistryCacheCapacity = 1000
	defaultMaxRetryAttempts            = 3
	defaultInitialBackoff              = 1 * time.Second
	defaultMaxBackoff                  = 30 * time.Second
	defaultChannelBufferSize           = 100
	defaultConsumerReadinessTimeout    = 60
	defaultProducerReadinessTimeout    = 30

	// Validation bounds
	minMaxRetryAttempts    = 1
	maxMaxRetryAttempts    = 100
	minInitialBackoff      = 100 * time.Millisecond
	maxInitialBackoff      = 30 * time.Second
	minMaxBackoff          = 1 * time.Second
	maxMaxBackoffDuration  = 5 * time.Minute
	minChannelBufferSize   = 10
	maxChannelBufferSize   = 10000
	minSchemaCacheCapacity = 100
	maxSchemaCacheCapacity = 100000
	maxReadinessTimeout    = 600 // 10 minutes in seconds
)

// Config represents the main Kafka configuration
type Config struct {
	Brokers         string               `mapstructure:"brokers"`          // Comma-separated list of Kafka broker addresses (e.g., "localhost:9092,localhost:9093")
	SchemaRegistry  SchemaRegistryConfig `mapstructure:"schema-registry"`  // Schema Registry configuration for Avro serialization/deserialization
	ConsumersConfig ConsumersConfig      `mapstructure:"consumers-config"` // Global and individual consumer configurations
	ProducerConfig  ProducerConfig       `mapstructure:"producer-config"`  // Producer-specific configuration
}

// ConsumersConfig holds global default settings and individual consumer configurations
type ConsumersConfig struct {
	DefaultGroupID           string           `mapstructure:"default-group-id"`            // Default consumer group ID (applied to consumers without explicit group-id)
	DefaultAutoOffsetReset   string           `mapstructure:"default-auto-offset-reset"`   // Default offset reset policy: "earliest" or "latest"
	DefaultMaxRetryAttempts  int              `mapstructure:"default-max-retry-attempts"`  // Default maximum retry attempts for message processing (1-100)
	DefaultInitialBackoff    time.Duration    `mapstructure:"default-initial-backoff"`     // Default initial backoff duration for retries (100ms-30s)
	DefaultMaxBackoff        time.Duration    `mapstructure:"default-max-backoff"`         // Default maximum backoff duration for retries (1s-5m)
	DefaultChannelBufferSize int              `mapstructure:"default-channel-buffer-size"` // Default internal message channel buffer size (10-10000)
	ConsumerConfig           []ConsumerConfig `mapstructure:"consumers"`                   // Individual consumer configurations
}

// ConsumerConfig represents configuration for an individual Kafka consumer
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
	ChannelBufferSize       int           `mapstructure:"channel-buffer-size"`       // Internal message channel buffer size (10-10000, defaults to DefaultChannelBufferSize)
}

// ProducerConfig represents configuration for Kafka producer
type ProducerConfig struct {
	ReadinessTimeoutSeconds int  `mapstructure:"readiness-timeout-seconds"` // Timeout in seconds for waiting brokers readiness (0 = no timeout, max 600s, default 30s)
	FailOnBrokerError       bool `mapstructure:"fail-on-broker-error"`      // Whether to fail application startup if brokers are not available (default false)
}

// SchemaRegistryConfig represents Confluent Schema Registry configuration
type SchemaRegistryConfig struct {
	URL                 string `mapstructure:"url"`                   // Schema Registry URL (required, e.g., "http://schema-registry:8081")
	CacheCapacity       int    `mapstructure:"cache-capacity"`        // Schema cache capacity (100-100000, default 1000)
	AutoRegisterSchemas bool   `mapstructure:"auto_register_schemas"` // Automatically register schemas on startup
}

func NewKafkaConfigModule() fx.Option {
	return fx.Provide(
		newConfig,
	)
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("kafka").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load kafka config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&cfg); err != nil {
		return cfg, err
	}

	// Apply defaults
	if cfg.SchemaRegistry.CacheCapacity == 0 {
		cfg.SchemaRegistry.CacheCapacity = defaultSchemaRegistryCacheCapacity
	}

	// Apply defaults for global consumer config
	if cfg.ConsumersConfig.DefaultMaxRetryAttempts == 0 {
		cfg.ConsumersConfig.DefaultMaxRetryAttempts = defaultMaxRetryAttempts
	}
	if cfg.ConsumersConfig.DefaultInitialBackoff == 0 {
		cfg.ConsumersConfig.DefaultInitialBackoff = defaultInitialBackoff
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff == 0 {
		cfg.ConsumersConfig.DefaultMaxBackoff = defaultMaxBackoff
	}
	if cfg.ConsumersConfig.DefaultChannelBufferSize == 0 {
		cfg.ConsumersConfig.DefaultChannelBufferSize = defaultChannelBufferSize
	}

	// Apply defaults from global consumer config to individual consumers
	for i := range cfg.ConsumersConfig.ConsumerConfig {
		consumer := &cfg.ConsumersConfig.ConsumerConfig[i]
		if consumer.GroupID == "" {
			consumer.GroupID = cfg.ConsumersConfig.DefaultGroupID
		}
		if consumer.AutoOffsetReset == "" {
			consumer.AutoOffsetReset = cfg.ConsumersConfig.DefaultAutoOffsetReset
		}
		// Apply default DLQ topic naming convention: {topic}.dlq
		if consumer.EnableDLQ && consumer.DLQTopic == "" {
			consumer.DLQTopic = consumer.Topic + ".dlq"
		}
		// Apply default readiness timeout
		if consumer.ReadinessTimeoutSeconds == 0 {
			consumer.ReadinessTimeoutSeconds = defaultConsumerReadinessTimeout
		}
		// Apply default max retry attempts from global config
		if consumer.MaxRetryAttempts == 0 {
			consumer.MaxRetryAttempts = cfg.ConsumersConfig.DefaultMaxRetryAttempts
		}
		// Apply default initial backoff from global config
		if consumer.InitialBackoff == 0 {
			consumer.InitialBackoff = cfg.ConsumersConfig.DefaultInitialBackoff
		}
		// Apply default max backoff from global config
		if consumer.MaxBackoff == 0 {
			consumer.MaxBackoff = cfg.ConsumersConfig.DefaultMaxBackoff
		}
		// Apply default channel buffer size from global config
		if consumer.ChannelBufferSize == 0 {
			consumer.ChannelBufferSize = cfg.ConsumersConfig.DefaultChannelBufferSize
		}
	}

	// Apply default producer config settings
	if cfg.ProducerConfig.ReadinessTimeoutSeconds == 0 {
		cfg.ProducerConfig.ReadinessTimeoutSeconds = defaultProducerReadinessTimeout
	}

	logger.Info("loaded kafka config")
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	// Validate brokers
	if strings.TrimSpace(cfg.Brokers) == "" {
		return fmt.Errorf("kafka brokers cannot be empty")
	}

	// Validate schema registry URL
	if strings.TrimSpace(cfg.SchemaRegistry.URL) == "" {
		return fmt.Errorf("schema registry URL cannot be empty")
	}

	// Validate schema registry cache capacity
	if cfg.SchemaRegistry.CacheCapacity < minSchemaCacheCapacity || cfg.SchemaRegistry.CacheCapacity > maxSchemaCacheCapacity {
		return fmt.Errorf("schema registry cache capacity must be between %d and %d, got: %d", minSchemaCacheCapacity, maxSchemaCacheCapacity, cfg.SchemaRegistry.CacheCapacity)
	}

	// Validate global consumer config
	if cfg.ConsumersConfig.DefaultMaxRetryAttempts > 0 && (cfg.ConsumersConfig.DefaultMaxRetryAttempts < minMaxRetryAttempts || cfg.ConsumersConfig.DefaultMaxRetryAttempts > maxMaxRetryAttempts) {
		return fmt.Errorf("default max retry attempts must be between %d and %d, got: %d", minMaxRetryAttempts, maxMaxRetryAttempts, cfg.ConsumersConfig.DefaultMaxRetryAttempts)
	}
	if cfg.ConsumersConfig.DefaultInitialBackoff > 0 && (cfg.ConsumersConfig.DefaultInitialBackoff < minInitialBackoff || cfg.ConsumersConfig.DefaultInitialBackoff > maxInitialBackoff) {
		return fmt.Errorf("default initial backoff must be between %v and %v, got: %v", minInitialBackoff, maxInitialBackoff, cfg.ConsumersConfig.DefaultInitialBackoff)
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff > 0 && (cfg.ConsumersConfig.DefaultMaxBackoff < minMaxBackoff || cfg.ConsumersConfig.DefaultMaxBackoff > maxMaxBackoffDuration) {
		return fmt.Errorf("default max backoff must be between %v and %v, got: %v", minMaxBackoff, maxMaxBackoffDuration, cfg.ConsumersConfig.DefaultMaxBackoff)
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff > 0 && cfg.ConsumersConfig.DefaultInitialBackoff > cfg.ConsumersConfig.DefaultMaxBackoff {
		return fmt.Errorf("default initial backoff (%v) cannot be greater than max backoff (%v)", cfg.ConsumersConfig.DefaultInitialBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	}
	if cfg.ConsumersConfig.DefaultChannelBufferSize > 0 && (cfg.ConsumersConfig.DefaultChannelBufferSize < minChannelBufferSize || cfg.ConsumersConfig.DefaultChannelBufferSize > maxChannelBufferSize) {
		return fmt.Errorf("default channel buffer size must be between %d and %d, got: %d", minChannelBufferSize, maxChannelBufferSize, cfg.ConsumersConfig.DefaultChannelBufferSize)
	}

	// Validate individual consumers
	for i, consumer := range cfg.ConsumersConfig.ConsumerConfig {
		if strings.TrimSpace(consumer.Name) == "" {
			return fmt.Errorf("consumer[%d]: name cannot be empty", i)
		}
		if strings.TrimSpace(consumer.Topic) == "" {
			return fmt.Errorf("consumer[%d] (%s): topic cannot be empty", i, consumer.Name)
		}
		if consumer.AutoOffsetReset != "" && consumer.AutoOffsetReset != "earliest" && consumer.AutoOffsetReset != "latest" {
			return fmt.Errorf("consumer[%d] (%s): auto offset reset must be 'earliest' or 'latest', got: %s", i, consumer.Name, consumer.AutoOffsetReset)
		}
		if consumer.ReadinessTimeoutSeconds > maxReadinessTimeout {
			return fmt.Errorf("consumer[%d] (%s): readiness timeout cannot exceed %d seconds, got: %d", i, consumer.Name, maxReadinessTimeout, consumer.ReadinessTimeoutSeconds)
		}
		if consumer.MaxRetryAttempts > 0 && (consumer.MaxRetryAttempts < minMaxRetryAttempts || consumer.MaxRetryAttempts > maxMaxRetryAttempts) {
			return fmt.Errorf("consumer[%d] (%s): max retry attempts must be between %d and %d, got: %d", i, consumer.Name, minMaxRetryAttempts, maxMaxRetryAttempts, consumer.MaxRetryAttempts)
		}
		if consumer.InitialBackoff > 0 && (consumer.InitialBackoff < minInitialBackoff || consumer.InitialBackoff > maxInitialBackoff) {
			return fmt.Errorf("consumer[%d] (%s): initial backoff must be between %v and %v, got: %v", i, consumer.Name, minInitialBackoff, maxInitialBackoff, consumer.InitialBackoff)
		}
		if consumer.MaxBackoff > 0 && (consumer.MaxBackoff < minMaxBackoff || consumer.MaxBackoff > maxMaxBackoffDuration) {
			return fmt.Errorf("consumer[%d] (%s): max backoff must be between %v and %v, got: %v", i, consumer.Name, minMaxBackoff, maxMaxBackoffDuration, consumer.MaxBackoff)
		}
		if consumer.MaxBackoff > 0 && consumer.InitialBackoff > consumer.MaxBackoff {
			return fmt.Errorf("consumer[%d] (%s): initial backoff (%v) cannot be greater than max backoff (%v)", i, consumer.Name, consumer.InitialBackoff, consumer.MaxBackoff)
		}
		if consumer.ChannelBufferSize > 0 && (consumer.ChannelBufferSize < minChannelBufferSize || consumer.ChannelBufferSize > maxChannelBufferSize) {
			return fmt.Errorf("consumer[%d] (%s): channel buffer size must be between %d and %d, got: %d", i, consumer.Name, minChannelBufferSize, maxChannelBufferSize, consumer.ChannelBufferSize)
		}
		if consumer.EnableDLQ && strings.TrimSpace(consumer.DLQTopic) != "" && consumer.DLQTopic == consumer.Topic {
			return fmt.Errorf("consumer[%d] (%s): DLQ topic cannot be the same as main topic", i, consumer.Name)
		}
	}

	// Validate producer config
	if cfg.ProducerConfig.ReadinessTimeoutSeconds > maxReadinessTimeout {
		return fmt.Errorf("producer readiness timeout cannot exceed %d seconds, got: %d", maxReadinessTimeout, cfg.ProducerConfig.ReadinessTimeoutSeconds)
	}

	return nil
}
