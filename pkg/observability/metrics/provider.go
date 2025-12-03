package metrics

import (
	"context"
	"fmt"
	"time"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	otelinternal "github.com/Sokol111/ecommerce-commons/pkg/observability/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// newProvider creates a new metrics Provider.
func newProvider(ctx context.Context, log *zap.Logger, endpoint string, interval time.Duration, appCfg appconfig.AppConfig) (*sdkmetric.MeterProvider, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("metrics: otel-collector-endpoint is required")
	}

	res, err := otelinternal.NewResource(ctx, appCfg)
	if err != nil {
		return nil, err
	}

	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	), nil
}
