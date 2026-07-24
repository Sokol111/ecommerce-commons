// Package fxconfig provides OpenTelemetry tracing, metrics, and continuous profiling integration for fx.
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
package fxconfig

import (
	"connectrpc.com/connect"
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/observability"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// observabilityOptions holds internal configuration for the observability module.
type observabilityOptions struct {
	config *observability.Config
}

// Option is a functional option for configuring the observability module.
type Option func(*observabilityOptions)

// WithConfig provides a static observability Config (useful for tests).
// When set, the observability configuration will not be loaded from koanf.
func WithConfig(cfg observability.Config) Option {
	return func(opts *observabilityOptions) {
		opts.config = &cfg
	}
}

// NewObservabilityModule returns fx.Option with full observability: tracing and metrics.
//
// Options:
//   - WithConfig: provide static observability Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from koanf
//	observability.NewObservabilityModule()
//
//	// Testing - disable observability
//	observability.NewObservabilityModule(
//	    observability.WithConfig(observability.Config{
//	        Tracing: observability.TracingConfig{Enabled: false},
//	        Metrics: observability.MetricsConfig{Enabled: false},
//	    }),
//	)
func NewObservabilityModule(opts ...Option) fx.Option {
	cfg := &observabilityOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Supply(
			fx.Annotate(
				fx_interceptor.Interceptor{Priority: 15, Handler: connect.UnaryInterceptorFunc(observability.TraceContextUnaryInterceptor)},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Provide(
			provideConfig,
			provideTracerProvider,
			provideMeterProvider,
		),
		fx.Invoke(
			startProfiler,
			activateTracing,
			activateMetrics,
		),
	)
}

func provideConfig(opts *observabilityOptions, loader *coreconfig.Loader, logger *zap.Logger) (observability.Config, error) {
	return coreconfig.Load[observability.Config](loader, "observability", opts.config)
}
