package tracing

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	otelinternal "github.com/Sokol111/ecommerce-commons/pkg/observability/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

// newTracerProvider creates a new OpenTelemetry TracerProvider.
func newTracerProvider(ctx context.Context, log *zap.Logger, endpoint string, appCfg appconfig.AppConfig) (*sdktrace.TracerProvider, error) {
	res, err := otelinternal.NewResource(ctx, appCfg)
	if err != nil {
		return nil, err
	}

	if endpoint == "" {
		log.Info("tracing: no collector endpoint, running in local mode")
		return sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
			sdktrace.WithResource(res),
		), nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	), nil
}
