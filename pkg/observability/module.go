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
package observability

import (
	"github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/metrics"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/tracing"
	"go.uber.org/fx"
)

// NewObservabilityModule returns fx.Option with full observability: tracing and metrics.
func NewObservabilityModule() fx.Option {
	return fx.Options(
		config.NewObservabilityConfigModule(),
		tracing.NewTracingModule(),
		metrics.NewMetricsModule(),
	)
}
