package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Brokers         string               `mapstructure:"brokers"`
	SchemaRegistry  SchemaRegistryConfig `mapstructure:"schema-registry"`
	ConsumersConfig ConsumersConfig      `mapstructure:"consumers-config"`
	ProducerConfig  ProducerConfig       `mapstructure:"producer-config"`
}

type ConsumersConfig struct {
	DefaultGroupID           string           `mapstructure:"default-group-id"`
	DefaultAutoOffsetReset   string           `mapstructure:"default-auto-offset-reset"`
	DefaultMaxRetryAttempts  int              `mapstructure:"default-max-retry-attempts"`
	DefaultInitialBackoff    time.Duration    `mapstructure:"default-initial-backoff"`
	DefaultMaxBackoff        time.Duration    `mapstructure:"default-max-backoff"`
	DefaultChannelBufferSize int              `mapstructure:"default-channel-buffer-size"`
	ConsumerConfig           []ConsumerConfig `mapstructure:"consumers"`
}

type ConsumerConfig struct {
	Name                    string        `mapstructure:"name"`
	Topic                   string        `mapstructure:"topic"`
	GroupID                 string        `mapstructure:"group-id"`
	AutoOffsetReset         string        `mapstructure:"auto-offset-reset"`
	EnableDLQ               bool          `mapstructure:"enable-dlq"`
	DLQTopic                string        `mapstructure:"dlq-topic"`
	ReadinessTimeoutSeconds int           `mapstructure:"readiness-timeout-seconds"` // Timeout for waiting topic readiness (0 = no timeout)
	FailOnTopicError        bool          `mapstructure:"fail-on-topic-error"`       // Whether to fail startup if topic is not available
	MaxRetryAttempts        int           `mapstructure:"max-retry-attempts"`        // Maximum number of retry attempts for message processing
	InitialBackoff          time.Duration `mapstructure:"initial-backoff"`           // Initial backoff duration between retries
	MaxBackoff              time.Duration `mapstructure:"max-backoff"`               // Maximum backoff duration between retries
	ChannelBufferSize       int           `mapstructure:"channel-buffer-size"`       // Size of the internal message channel buffer
}

type ProducerConfig struct {
	ReadinessTimeoutSeconds int  `mapstructure:"readiness-timeout-seconds"` // Timeout for waiting brokers readiness (0 = no timeout)
	FailOnBrokerError       bool `mapstructure:"fail-on-broker-error"`      // Whether to fail startup if brokers are not available
}

type SchemaRegistryConfig struct {
	URL                 string `mapstructure:"url"`
	CacheCapacity       int    `mapstructure:"cache-capacity"`
	AutoRegisterSchemas bool   `mapstructure:"auto_register_schemas"`
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
		cfg.SchemaRegistry.CacheCapacity = 1000
	}

	// Apply defaults for global consumer config
	if cfg.ConsumersConfig.DefaultMaxRetryAttempts == 0 {
		cfg.ConsumersConfig.DefaultMaxRetryAttempts = 5
	}
	if cfg.ConsumersConfig.DefaultInitialBackoff == 0 {
		cfg.ConsumersConfig.DefaultInitialBackoff = 1 * time.Second
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff == 0 {
		cfg.ConsumersConfig.DefaultMaxBackoff = 10 * time.Second
	}
	if cfg.ConsumersConfig.DefaultChannelBufferSize == 0 {
		cfg.ConsumersConfig.DefaultChannelBufferSize = 100
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
		// Apply default readiness timeout: 60 seconds
		if consumer.ReadinessTimeoutSeconds == 0 {
			consumer.ReadinessTimeoutSeconds = 60
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
		cfg.ProducerConfig.ReadinessTimeoutSeconds = 60
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
	if cfg.SchemaRegistry.CacheCapacity < 0 {
		return fmt.Errorf("schema registry cache capacity cannot be negative, got: %d", cfg.SchemaRegistry.CacheCapacity)
	}

	// Validate global consumer config
	if cfg.ConsumersConfig.DefaultMaxRetryAttempts < 0 {
		return fmt.Errorf("default max retry attempts cannot be negative, got: %d", cfg.ConsumersConfig.DefaultMaxRetryAttempts)
	}
	if cfg.ConsumersConfig.DefaultInitialBackoff < 0 {
		return fmt.Errorf("default initial backoff cannot be negative, got: %v", cfg.ConsumersConfig.DefaultInitialBackoff)
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff < 0 {
		return fmt.Errorf("default max backoff cannot be negative, got: %v", cfg.ConsumersConfig.DefaultMaxBackoff)
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff > 0 && cfg.ConsumersConfig.DefaultInitialBackoff > cfg.ConsumersConfig.DefaultMaxBackoff {
		return fmt.Errorf("default initial backoff (%v) cannot be greater than max backoff (%v)", cfg.ConsumersConfig.DefaultInitialBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	}
	if cfg.ConsumersConfig.DefaultChannelBufferSize < 0 {
		return fmt.Errorf("default channel buffer size cannot be negative, got: %d", cfg.ConsumersConfig.DefaultChannelBufferSize)
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
		if consumer.ReadinessTimeoutSeconds < 0 {
			return fmt.Errorf("consumer[%d] (%s): readiness timeout cannot be negative, got: %d", i, consumer.Name, consumer.ReadinessTimeoutSeconds)
		}
		if consumer.MaxRetryAttempts < 0 {
			return fmt.Errorf("consumer[%d] (%s): max retry attempts cannot be negative, got: %d", i, consumer.Name, consumer.MaxRetryAttempts)
		}
		if consumer.InitialBackoff < 0 {
			return fmt.Errorf("consumer[%d] (%s): initial backoff cannot be negative, got: %v", i, consumer.Name, consumer.InitialBackoff)
		}
		if consumer.MaxBackoff < 0 {
			return fmt.Errorf("consumer[%d] (%s): max backoff cannot be negative, got: %v", i, consumer.Name, consumer.MaxBackoff)
		}
		if consumer.MaxBackoff > 0 && consumer.InitialBackoff > consumer.MaxBackoff {
			return fmt.Errorf("consumer[%d] (%s): initial backoff (%v) cannot be greater than max backoff (%v)", i, consumer.Name, consumer.InitialBackoff, consumer.MaxBackoff)
		}
		if consumer.ChannelBufferSize < 0 {
			return fmt.Errorf("consumer[%d] (%s): channel buffer size cannot be negative, got: %d", i, consumer.Name, consumer.ChannelBufferSize)
		}
		if consumer.EnableDLQ && strings.TrimSpace(consumer.DLQTopic) != "" && consumer.DLQTopic == consumer.Topic {
			return fmt.Errorf("consumer[%d] (%s): DLQ topic cannot be the same as main topic", i, consumer.Name)
		}
	}

	// Validate producer config
	if cfg.ProducerConfig.ReadinessTimeoutSeconds < 0 {
		return fmt.Errorf("producer readiness timeout cannot be negative, got: %d", cfg.ProducerConfig.ReadinessTimeoutSeconds)
	}

	return nil
}
