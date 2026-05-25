// Package observability provides OpenTelemetry tracing, metrics, and continuous profiling integration.
//
// Usage:
//
//	// Full observability (tracing + metrics + profiling)
//	observability.NewObservabilityModule()
//
//	// Only tracing (requires config.NewObservabilityConfigModule())
//	tracing.NewTracingModule()
//
//	// Only metrics (requires config.NewObservabilityConfigModule())
//	metrics.NewMetricsModule()
//
//	// Only profiling (requires config.NewObservabilityConfigModule())
//	profiling.NewProfilingModule()
//
//	// Disable observability for tests
//	observability.NewObservabilityModule(
//	    observability.WithoutTracing(),
//	    observability.WithoutMetrics(),
//	    observability.WithoutProfiling(),
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
	"github.com/Sokol111/ecommerce-commons/pkg/observability/profiling"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/tracing"
	"go.uber.org/fx"
)

// observabilityOptions holds internal configuration for the observability module.
type observabilityOptions struct {
	config           *config.Config
	disableTracing   bool
	disableMetrics   bool
	disableProfiling bool
}

// Option is a functional option for configuring the observability module.
type Option func(*observabilityOptions)

// WithConfig provides a static observability Config (useful for tests).
// When set, the observability configuration will not be loaded from koanf.
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

// WithoutProfiling disables continuous profiling regardless of configuration.
// Useful for tests where profiling is not needed.
func WithoutProfiling() Option {
	return func(opts *observabilityOptions) {
		opts.disableProfiling = true
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
//	// Production - loads config from koanf
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
		profiling.NewProfilingModule(),
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
	if opts.disableProfiling {
		configOpts = append(configOpts, config.WithDisableProfiling())
	}

	return config.NewObservabilityConfigModule(configOpts...)
}
