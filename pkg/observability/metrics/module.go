package metrics

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	otelconfig "github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// providerParams holds dependencies for metrics provider.
type providerParams struct {
	fx.In
	Lc        fx.Lifecycle
	Log       *zap.Logger
	Cfg       otelconfig.Config
	AppCfg    appconfig.AppConfig
	Readiness health.ComponentManager
}

// NewMetricsModule returns fx.Option for metrics.
//
// Note: HTTP request metrics are handled by ogen-generated servers automatically.
// This module only provides the MeterProvider and runtime metrics.
func NewMetricsModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(p providerParams) (metric.MeterProvider, error) {
				if !p.Cfg.Metrics.Enabled {
					p.Log.Info("metrics: disabled")
					return noop.NewMeterProvider(), nil
				}
				return provideMeterProvider(p)
			},
		),
		fx.Invoke(func(metric.MeterProvider) {}),
	)
}

func provideMeterProvider(p providerParams) (metric.MeterProvider, error) {
	markReady := p.Readiness.AddComponent(otelconfig.MetricsComponentName)

	provider, err := newProvider(context.Background(), p.Cfg.OtelCollectorEndpoint, p.Cfg.Metrics.Interval, p.AppCfg)
	if err != nil {
		return nil, err
	}

	p.Lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetMeterProvider(provider)
			_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(otelconfig.DefaultRuntimeStatsInterval)) //nolint:errcheck // best-effort runtime stats
			p.Log.Info("metrics initialized",
				zap.String("endpoint", p.Cfg.OtelCollectorEndpoint),
				zap.Duration("interval", p.Cfg.Metrics.Interval),
			)
			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, otelconfig.DefaultShutdownTimeout)
			defer cancel()
			return provider.Shutdown(shutdownCtx)
		},
	})

	return provider, nil
}
