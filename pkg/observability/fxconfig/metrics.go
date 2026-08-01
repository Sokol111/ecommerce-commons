package fxconfig

import (
	"context"
	"fmt"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/observability"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// providerParams holds dependencies for metrics provider.
type providerParams struct {
	fx.In
	Log        *zap.Logger
	Cfg        observability.Config
	AppCfg     appconfig.AppConfig
	ExtraViews []sdkmetric.View `group:"metric_views"`
}

// provideMeterProvider creates a new metrics Provider.
func provideMeterProvider(p providerParams) (metric.MeterProvider, *sdkmetric.MeterProvider, error) {
	if !p.Cfg.Metrics.Enabled {
		p.Log.Info("metrics: disabled")
		return noop.NewMeterProvider(), nil, nil
	}
	if p.Cfg.OtelCollectorEndpoint == "" {
		return nil, nil, fmt.Errorf("metrics: otel-collector-endpoint is required")
	}

	ctx := context.Background()

	res, err := newResource(ctx, p.AppCfg)
	if err != nil {
		return nil, nil, err
	}

	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(p.Cfg.OtelCollectorEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	allViews := append(metricViews(), p.ExtraViews...)

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(p.Cfg.Metrics.Interval))
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
		sdkmetric.WithView(allViews...),
	)
	return provider, provider, nil
}

// metricViews returns SDK views to reduce cardinality of runtime metrics.
func metricViews() []sdkmetric.View {
	return []sdkmetric.View{
		// Drop all go.schedule.duration — rarely useful, high bucket count.
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "go.schedule.duration"},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationDrop{}},
		),
		// Drop go.memory.limit — constant value, not useful.
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "go.memory.limit"},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationDrop{}},
		),
	}
}

func activateMetrics(lc fx.Lifecycle, log *zap.Logger, cfg observability.Config, provider *sdkmetric.MeterProvider, readiness health.ComponentManager) {
	markReady := readiness.AddComponent(observability.MetricsComponentName)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if provider == nil {
				log.Info("metrics: disabled, skipping activation")
				markReady()
				return nil
			}
			otel.SetMeterProvider(provider)
			_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(observability.DefaultRuntimeStatsInterval)) //nolint:errcheck // best-effort runtime stats
			log.Info("metrics initialized",
				zap.String("endpoint", cfg.OtelCollectorEndpoint),
				zap.Duration("interval", cfg.Metrics.Interval),
			)
			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if provider == nil {
				return nil
			}
			shutdownCtx, cancel := context.WithTimeout(ctx, observability.DefaultShutdownTimeout)
			defer cancel()
			return provider.Shutdown(shutdownCtx)
		},
	})
}
