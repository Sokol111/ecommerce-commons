package tracing

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// providerParams holds dependencies for tracing provider.
type providerParams struct {
	fx.In
	Lc        fx.Lifecycle
	Log       *zap.Logger
	Cfg       config.Config
	AppCfg    appconfig.AppConfig
	Readiness health.ComponentManager
}

// NewTracingModule returns fx.Option for tracing.
// Provides TracerProvider and configures OpenTelemetry globals.
func NewTracingModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(p providerParams) (trace.TracerProvider, error) {
				if !p.Cfg.Tracing.Enabled {
					p.Log.Info("tracing: disabled")
					return noop.NewTracerProvider(), nil
				}
				return provideTracerProvider(p)
			},
		),
		fx.Invoke(func(trace.TracerProvider) {}),
	)
}

func provideTracerProvider(p providerParams) (trace.TracerProvider, error) {
	tp, err := newTracerProvider(context.Background(), p.Log, p.Cfg, p.AppCfg)
	if err != nil {
		return nil, err
	}

	markReady := p.Readiness.AddComponent(config.TracingComponentName)

	p.Lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			p.Log.Info("tracing initialized", zap.String("endpoint", p.Cfg.OtelCollectorEndpoint))
			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, config.DefaultShutdownTimeout)
			defer cancel()
			return tp.Shutdown(shutdownCtx)
		},
	})

	return tp, nil
}
