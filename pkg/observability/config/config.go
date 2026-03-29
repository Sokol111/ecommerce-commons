package config

import "time"

const (
	// DefaultMetricsInterval is the default metrics collection interval.
	DefaultMetricsInterval = 10 * time.Second

	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 5 * time.Second

	// DefaultRuntimeStatsInterval is the default interval for runtime stats.
	DefaultRuntimeStatsInterval = 10 * time.Second

	// TracingComponentName is the name used for health check registration.
	TracingComponentName = "tracing"

	// MetricsComponentName is the name used for health check registration.
	MetricsComponentName = "metrics"
)

// Config holds all observability configuration.
type Config struct {
	OtelCollectorEndpoint string        `koanf:"otel-collector-endpoint"`
	Tracing               TracingConfig `koanf:"tracing"`
	Metrics               MetricsConfig `koanf:"metrics"`
}

// TracingConfig holds tracing-specific configuration.
type TracingConfig struct {
	Enabled     bool    `koanf:"enabled"`
	SampleRatio float64 `koanf:"sample-ratio"`
}

const (
	// DefaultSampleRatio is the default sampling ratio (100% for local development).
	DefaultSampleRatio = 1.0
)

// MetricsConfig holds metrics-specific configuration.
type MetricsConfig struct {
	Enabled  bool          `koanf:"enabled"`
	Interval time.Duration `koanf:"interval"`
}
