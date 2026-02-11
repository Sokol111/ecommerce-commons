package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// configOptions holds internal configuration for the observability config module.
type configOptions struct {
	config         *Config
	disableTracing bool
	disableMetrics bool
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

// NewObservabilityConfigModule provides observability configuration.
// By default, configuration is loaded from viper.
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

func provideConfig(opts *configOptions, v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if opts.config != nil {
		cfg = *opts.config
	} else if sub := v.Sub("observability"); sub != nil {
		if err := sub.Unmarshal(&cfg); err != nil {
			return cfg, fmt.Errorf("failed to load observability config: %w", err)
		}
	}

	applyDefaults(&cfg)
	applyDisableOptions(&cfg, opts)

	logger.Info("loaded observability config")
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Metrics.Interval == 0 {
		cfg.Metrics.Interval = DefaultMetricsInterval
	}
	if cfg.Tracing.SampleRatio == 0 {
		cfg.Tracing.SampleRatio = DefaultSampleRatio
	}
}

func applyDisableOptions(cfg *Config, opts *configOptions) {
	if opts.disableTracing {
		cfg.Tracing.Enabled = false
	}
	if opts.disableMetrics {
		cfg.Metrics.Enabled = false
	}
}
