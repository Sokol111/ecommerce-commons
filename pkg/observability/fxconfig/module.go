// Package fxconfig provides Fx wiring for OpenTelemetry tracing, metrics, and continuous profiling.
package fxconfig

import (
	"connectrpc.com/connect"
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/observability"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewObservabilityModule returns Fx wiring for tracing, metrics, and profiling.
// Configuration is loaded from the "observability" config section.
func NewObservabilityModule() fx.Option {
	return fx.Options(
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

func provideConfig(loader *coreconfig.Loader, logger *zap.Logger) (observability.Config, error) {
	return coreconfig.Load[observability.Config](loader, "observability", nil)
}
