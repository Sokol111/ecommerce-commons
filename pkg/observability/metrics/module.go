package metrics

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	otelconfig "github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	otelinternal "github.com/Sokol111/ecommerce-commons/pkg/observability/internal"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
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
func NewMetricsModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(p providerParams) (metric.MeterProvider, error) {
				if !p.Cfg.Metrics.Enabled {
					p.Log.Info("metrics: disabled")
					return nil, nil
				}
				return provideMeterProvider(p)
			},
			fx.Annotate(
				func(appCfg appconfig.AppConfig, mp metric.MeterProvider) middleware.Middleware {
					if mp == nil {
						return middleware.Middleware{}
					}
					return httpMiddleware(appCfg, mp)
				},
				fx.ResultTags(`group:"gin_mw"`),
			),
		),
		fx.Invoke(func(metric.MeterProvider) {}),
	)
}

func provideMeterProvider(p providerParams) (metric.MeterProvider, error) {
	p.Readiness.AddComponent(otelconfig.MetricsComponentName)

	provider, err := newProvider(context.Background(), p.Log, p.Cfg.OtelCollectorEndpoint, p.Cfg.Metrics.Interval, p.AppCfg)
	if err != nil {
		return nil, err
	}

	p.Lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetMeterProvider(provider)
			_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(otelconfig.DefaultRuntimeStatsInterval))
			p.Log.Info("metrics initialized",
				zap.String("endpoint", p.Cfg.OtelCollectorEndpoint),
				zap.Duration("interval", p.Cfg.Metrics.Interval),
			)
			p.Readiness.MarkReady(otelconfig.MetricsComponentName)
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

func httpMiddleware(appCfg appconfig.AppConfig, mp metric.MeterProvider) middleware.Middleware {
	return middleware.Middleware{
		Priority: 6, // After tracing middleware (5)
		Handler: otelgin.Middleware(appCfg.ServiceName,
			otelgin.WithMeterProvider(mp),
			otelgin.WithGinFilter(otelinternal.FilterPaths),
		),
	}
}
