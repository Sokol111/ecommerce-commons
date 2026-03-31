package metrics

import (
	"context"
	"fmt"
	"time"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	otelinternal "github.com/Sokol111/ecommerce-commons/pkg/observability/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// newProvider creates a new metrics Provider.
func newProvider(ctx context.Context, endpoint string, interval time.Duration, appCfg appconfig.AppConfig, extraViews []sdkmetric.View) (*sdkmetric.MeterProvider, error) {
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

	allViews := append(metricViews(), extraViews...)

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
		sdkmetric.WithView(allViews...),
	), nil
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
