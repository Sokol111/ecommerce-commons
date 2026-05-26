package config

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// configOptions holds internal configuration for the observability config module.
type configOptions struct {
	config           *Config
	disableTracing   bool
	disableMetrics   bool
	disableProfiling bool
}

// Option is a functional option for configuring the observability config module.
type Option func(*configOptions)

// WithConfig provides a static Config (useful for tests).
func WithConfig(cfg Config) Option {
	return func(opts *configOptions) {
		opts.config = &cfg
	}
}

// WithDisableTracing disables tracing regardless of configuration.
func WithDisableTracing() Option {
	return func(opts *configOptions) {
		opts.disableTracing = true
	}
}

// WithDisableMetrics disables metrics regardless of configuration.
func WithDisableMetrics() Option {
	return func(opts *configOptions) {
		opts.disableMetrics = true
	}
}

// WithDisableProfiling disables profiling regardless of configuration.
func WithDisableProfiling() Option {
	return func(opts *configOptions) {
		opts.disableProfiling = true
	}
}

// NewObservabilityConfigModule provides observability configuration.
// By default, configuration is loaded from koanf.
// Use WithConfig for static config (useful for tests).
func NewObservabilityConfigModule(opts ...Option) fx.Option {
	cfg := &configOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(provideConfig),
	)
}

func provideConfig(opts *configOptions, k *koanf.Koanf, logger *zap.Logger) (Config, error) {
	cfg, err := coreconfig.Load[Config](k, "observability", opts.config)
	if err != nil {
		return cfg, err
	}

	applyDisableOptions(&cfg, opts)

	logger.Info("loaded observability config")
	return cfg, nil
}

func applyDisableOptions(cfg *Config, opts *configOptions) {
	if opts.disableTracing {
		cfg.Tracing.Enabled = false
	}
	if opts.disableMetrics {
		cfg.Metrics.Enabled = false
	}
	if opts.disableProfiling {
		cfg.Profiling.Enabled = false
	}
}
