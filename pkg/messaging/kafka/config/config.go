package config

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/v2"
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
// By default, configuration is loaded from koanf.
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

func provideConfig(opts *kafkaConfigOptions, k *koanf.Koanf, logger *zap.Logger) (Config, error) {
	cfg, err := coreconfig.Load[Config](k, "kafka", opts.config)
	if err != nil {
		return Config{}, err
	}

	logger.Info("loaded kafka config")
	return cfg, nil
}
