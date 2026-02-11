// Package observability provides OpenTelemetry tracing and metrics integration.
//
// Usage:
//
//	// Full observability (tracing + metrics)
//	observability.NewObservabilityModule()
//
//	// Only tracing (requires config.NewObservabilityConfigModule())
//	tracing.NewTracingModule()
//
//	// Only metrics (requires config.NewObservabilityConfigModule())
//	metrics.NewMetricsModule()
//
//	// Disable observability for tests
//	observability.NewObservabilityModule(
//	    observability.WithoutTracing(),
//	    observability.WithoutMetrics(),
//	)
//
//	// Disable only tracing
//	observability.NewObservabilityModule(
//	    observability.WithoutTracing(),
//	)
package observability

import (
	"github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/metrics"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/tracing"
	"go.uber.org/fx"
)

// observabilityOptions holds internal configuration for the observability module.
type observabilityOptions struct {
	config         *config.Config
	disableTracing bool
	disableMetrics bool
}

// Option is a functional option for configuring the observability module.
type Option func(*observabilityOptions)

// WithConfig provides a static observability Config (useful for tests).
// When set, the observability configuration will not be loaded from viper.
func WithConfig(cfg config.Config) Option {
	return func(opts *observabilityOptions) {
		opts.config = &cfg
	}
}

// WithoutTracing disables tracing regardless of configuration.
// Useful for tests where tracing is not needed.
func WithoutTracing() Option {
	return func(opts *observabilityOptions) {
		opts.disableTracing = true
	}
}

// WithoutMetrics disables metrics regardless of configuration.
// Useful for tests where metrics are not needed.
func WithoutMetrics() Option {
	return func(opts *observabilityOptions) {
		opts.disableMetrics = true
	}
}

// NewObservabilityModule returns fx.Option with full observability: tracing and metrics.
//
// Options:
//   - WithConfig: provide static observability Config (useful for tests)
//   - WithoutTracing: disable tracing regardless of configuration
//   - WithoutMetrics: disable metrics regardless of configuration
//
// Example usage:
//
//	// Production - loads config from viper
//	observability.NewObservabilityModule()
//
//	// Testing - disable observability
//	observability.NewObservabilityModule(
//	    observability.WithoutTracing(),
//	    observability.WithoutMetrics(),
//	)
func NewObservabilityModule(opts ...Option) fx.Option {
	cfg := &observabilityOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		configModule(cfg),
		tracing.NewTracingModule(),
		metrics.NewMetricsModule(),
	)
}

func configModule(opts *observabilityOptions) fx.Option {
	var configOpts []config.Option

	if opts.config != nil {
		configOpts = append(configOpts, config.WithConfig(*opts.config))
	}
	if opts.disableTracing {
		configOpts = append(configOpts, config.WithDisableTracing())
	}
	if opts.disableMetrics {
		configOpts = append(configOpts, config.WithDisableMetrics())
	}

	return config.NewObservabilityConfigModule(configOpts...)
}
