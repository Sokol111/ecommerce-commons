package config

import (
	"fmt"

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
	GroupID         string           `mapstructure:"group-id"`
	AutoOffsetReset string           `mapstructure:"auto-offset-reset"`
	ConsumerConfig  []ConsumerConfig `mapstructure:"consumers"`
}

type ConsumerConfig struct {
	Name                    string `mapstructure:"name"`
	Topic                   string `mapstructure:"topic"`
	Subject                 string `mapstructure:"subject"`
	GroupID                 string `mapstructure:"group-id"`
	AutoOffsetReset         string `mapstructure:"auto-offset-reset"`
	EnableDLQ               bool   `mapstructure:"enable-dlq"`
	DLQTopic                string `mapstructure:"dlq-topic"`
	ReadinessTimeoutSeconds int    `mapstructure:"readiness-timeout-seconds"` // Timeout for waiting topic readiness (0 = no timeout)
	FailOnTopicError        bool   `mapstructure:"fail-on-topic-error"`       // Whether to fail startup if topic is not available
}

type ProducerConfig struct {
	ReadinessTimeoutSeconds int   `mapstructure:"readiness-timeout-seconds"` // Timeout for waiting brokers readiness (0 = no timeout)
	FailOnBrokerError       *bool `mapstructure:"fail-on-broker-error"`      // Whether to fail startup if brokers are not available
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

	if cfg.SchemaRegistry.CacheCapacity == 0 {
		cfg.SchemaRegistry.CacheCapacity = 1000
	}

	// Apply defaults from global consumer config to individual consumers
	for i := range cfg.ConsumersConfig.ConsumerConfig {
		consumer := &cfg.ConsumersConfig.ConsumerConfig[i]
		if consumer.GroupID == "" {
			consumer.GroupID = cfg.ConsumersConfig.GroupID
		}
		if consumer.AutoOffsetReset == "" {
			consumer.AutoOffsetReset = cfg.ConsumersConfig.AutoOffsetReset
		}
		// Apply default subject naming convention: {topic}-value
		if consumer.Subject == "" {
			consumer.Subject = consumer.Topic + "-value"
		}
		// Apply default DLQ topic naming convention: {topic}.dlq
		if consumer.EnableDLQ && consumer.DLQTopic == "" {
			consumer.DLQTopic = consumer.Topic + ".dlq"
		}
		// Apply default readiness timeout: 60 seconds
		if consumer.ReadinessTimeoutSeconds == 0 {
			consumer.ReadinessTimeoutSeconds = 60
		}
	}

	// Apply default producer config settings
	if cfg.ProducerConfig.ReadinessTimeoutSeconds == 0 {
		cfg.ProducerConfig.ReadinessTimeoutSeconds = 60
	}
	// Apply default fail-on-broker-error: true
	if cfg.ProducerConfig.FailOnBrokerError == nil {
		defaultFailOnBrokerError := true
		cfg.ProducerConfig.FailOnBrokerError = &defaultFailOnBrokerError
	}

	logger.Info("loaded kafka config", zap.Any("config", cfg))
	return cfg, nil
}
