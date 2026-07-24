package fxconfig

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// provideTracerProvider creates a new OpenTelemetry TracerProvider.
func provideTracerProvider(log *zap.Logger, cfg observability.Config, appCfg appconfig.AppConfig) (trace.TracerProvider, *sdktrace.TracerProvider, error) {
	if !cfg.Tracing.Enabled {
		log.Info("tracing: disabled")
		return noop.NewTracerProvider(), nil, nil
	}

	ctx := context.Background()
	res, err := newResource(ctx, appCfg)
	if err != nil {
		return nil, nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Tracing.SampleRatio))

	if cfg.OtelCollectorEndpoint == "" {
		log.Info("tracing: no collector endpoint, running in local mode",
			zap.Float64("sample_ratio", cfg.Tracing.SampleRatio))
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithResource(res),
		)
		return provider, provider, nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtelCollectorEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	return provider, provider, nil
}

func activateTracing(lc fx.Lifecycle, log *zap.Logger, provider *sdktrace.TracerProvider, cfg observability.Config, readiness health.ComponentManager) {
	if !cfg.Tracing.Enabled {
		return
	}
	markReady := readiness.AddComponent(observability.TracingComponentName)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			otel.SetTracerProvider(provider)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			log.Info("tracing initialized", zap.String("endpoint", cfg.OtelCollectorEndpoint))
			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, observability.DefaultShutdownTimeout)
			defer cancel()
			return provider.Shutdown(shutdownCtx)
		},
	})
}
