package internal

import (
	"context"
	"strings"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/gin-gonic/gin"
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

// FilterPaths returns a filter function that excludes specified paths from instrumentation.
func FilterPaths(c *gin.Context) bool {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	for _, excluded := range ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}
	return true
}
