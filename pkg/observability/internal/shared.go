package internal

import (
	"context"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// ExcludedPaths contains paths that should be excluded from tracing and metrics.
var ExcludedPaths = []string{"/health", "/metrics"}

// NewResource creates a new OpenTelemetry resource with service information.
func NewResource(ctx context.Context, appCfg appconfig.AppConfig) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(appCfg.ServiceName),
			semconv.ServiceVersionKey.String(appCfg.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(appCfg.Environment),
		),
	)
}
