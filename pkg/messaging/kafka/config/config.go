package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// kafkaConfigOptions holds internal configuration for the Kafka config module.
type kafkaConfigOptions struct {
	config *Config
}

// KafkaConfigOption is a functional option for configuring the Kafka config module.
type KafkaConfigOption func(*kafkaConfigOptions)

// WithKafkaConfig provides a static Config (useful for tests).
func WithKafkaConfig(cfg Config) KafkaConfigOption {
	return func(opts *kafkaConfigOptions) {
		opts.config = &cfg
	}
}

// NewKafkaConfigModule provides Kafka configuration for dependency injection.
// By default, configuration is loaded from viper.
// Use WithKafkaConfig for static config (useful for tests).
func NewKafkaConfigModule(opts ...KafkaConfigOption) fx.Option {
	cfg := &kafkaConfigOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(provideConfig),
	)
}

func provideConfig(opts *kafkaConfigOptions, v *viper.Viper, logger *zap.Logger) (Config, error) {
	var result Config
	if opts.config != nil {
		result = *opts.config
	} else {
		var err error
		result, err = loadFromViper(v)
		if err != nil {
			return result, err
		}
	}

	// Validate required fields
	if err := result.Validate(); err != nil {
		return result, err
	}

	// Apply default values
	result.ApplyDefaults()

	logger.Info("loaded kafka config")
	return result, nil
}

// loadFromViper loads Config from viper configuration.
func loadFromViper(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("kafka").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load kafka config: %w", err)
	}
	return cfg, nil
}
