package config

import "time"

const (
	// DefaultMetricsInterval is the default metrics collection interval.
	DefaultMetricsInterval = 10 * time.Second

	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 5 * time.Second

	// DefaultRuntimeStatsInterval is the default interval for runtime stats.
	DefaultRuntimeStatsInterval = time.Second

	// TracingComponentName is the name used for health check registration.
	TracingComponentName = "tracing"

	// MetricsComponentName is the name used for health check registration.
	MetricsComponentName = "metrics"
)

// Config holds all observability configuration.
type Config struct {
	OtelCollectorEndpoint string        `mapstructure:"otel-collector-endpoint"`
	Tracing               TracingConfig `mapstructure:"tracing"`
	Metrics               MetricsConfig `mapstructure:"metrics"`
}

// TracingConfig holds tracing-specific configuration.
type TracingConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// MetricsConfig holds metrics-specific configuration.
type MetricsConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
}
