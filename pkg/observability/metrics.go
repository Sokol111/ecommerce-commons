package observability

import (
	"context"
	"time"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"

	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"github.com/Sokol111/ecommerce-commons/pkg/health"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewMetricsModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config, readiness health.Readiness) (metric.MeterProvider, error) {
				if !conf.MetricsEnabled {
					log.Info("otel metrics: disabled")
					return nil, nil
				}
				return provideMeterProvider(lc, log, conf, appConf, readiness)
			},
		),
		fx.Invoke(func(metric.MeterProvider) {}),
	)
}

func provideMeterProvider(lc fx.Lifecycle, log *zap.Logger, conf Config, appConf config.Config, readiness health.Readiness) (metric.MeterProvider, error) {
	readiness.AddOne()
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(appConf.ServiceName),
			semconv.ServiceVersionKey.String(appConf.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(string(appConf.Environment)),
		),
	)
	if err != nil {
		return nil, err
	}

	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(conf.OtelCollectorEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	interval := conf.MetricsInterval
	if interval == 0 {
		interval = 10 * time.Second
	}
	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetMeterProvider(mp)

			_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(time.Second))

			log.Info("otel metrics initialized",
				zap.String("endpoint", conf.OtelCollectorEndpoint),
				zap.Duration("interval", interval),
			)
			readiness.Done()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return mp.Shutdown(c)
		},
	})

	return mp, nil
}
