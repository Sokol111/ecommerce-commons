package tracing

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	otelconfig "github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	otelinternal "github.com/Sokol111/ecommerce-commons/pkg/observability/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

// newTracerProvider creates a new OpenTelemetry TracerProvider.
func newTracerProvider(ctx context.Context, log *zap.Logger, cfg otelconfig.Config, appCfg appconfig.AppConfig) (*sdktrace.TracerProvider, error) {
	res, err := otelinternal.NewResource(ctx, appCfg)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Tracing.SampleRatio))

	if cfg.OtelCollectorEndpoint == "" {
		log.Info("tracing: no collector endpoint, running in local mode",
			zap.Float64("sample_ratio", cfg.Tracing.SampleRatio))
		return sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithResource(res),
		), nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtelCollectorEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	), nil
}
