package fxconfig

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	config "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// kafkaConfigOptions holds internal configuration for the Kafka config module.
type kafkaConfigOptions struct {
	config *config.Config
}

// Option is a functional option for configuring the Kafka config module.
type Option func(*kafkaConfigOptions)

// WithKafkaConfig provides a static Config (useful for tests).
func WithKafkaConfig(cfg config.Config) Option {
	return func(opts *kafkaConfigOptions) {
		opts.config = &cfg
	}
}

// NewKafkaConfigModule provides Kafka configuration for dependency injection.
// By default, configuration is loaded from koanf.
// Use WithKafkaConfig for static config (useful for tests).
func NewKafkaConfigModule(opts ...Option) fx.Option {
	cfg := &kafkaConfigOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(provideConfig),
	)
}

func provideConfig(opts *kafkaConfigOptions, loader *coreconfig.Loader, logger *zap.Logger) (config.Config, error) {
	return coreconfig.Load[config.Config](loader, "kafka", opts.config)
}
