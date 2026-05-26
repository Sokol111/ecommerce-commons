package config

import (
	"fmt"
	"time"
)

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
	OtelCollectorEndpoint string          `koanf:"otel-collector-endpoint"`
	Tracing               TracingConfig   `koanf:"tracing"`
	Metrics               MetricsConfig   `koanf:"metrics"`
	Profiling             ProfilingConfig `koanf:"profiling"`
}

// TracingConfig holds tracing-specific configuration.
type TracingConfig struct {
	Enabled     bool    `koanf:"enabled"`
	SampleRatio float64 `koanf:"sample-ratio"`
}

const (
	// DefaultSampleRatio is the default sampling ratio (100% for local development).
	DefaultSampleRatio = 1.0

	// DefaultMutexProfileFraction is the default mutex profile fraction.
	DefaultMutexProfileFraction = 5

	// DefaultBlockProfileRate is the default block profile rate.
	DefaultBlockProfileRate = 5
)

// MetricsConfig holds metrics-specific configuration.
type MetricsConfig struct {
	Enabled  bool          `koanf:"enabled"`
	Interval time.Duration `koanf:"interval"`
}

// ProfilingConfig holds continuous profiling configuration.
type ProfilingConfig struct {
	Enabled              bool   `koanf:"enabled"`
	Endpoint             string `koanf:"endpoint"`
	CPU                  *bool  `koanf:"cpu"`
	Heap                 *bool  `koanf:"heap"`
	Goroutines           *bool  `koanf:"goroutines"`
	Mutex                *bool  `koanf:"mutex"`
	Block                *bool  `koanf:"block"`
	MutexProfileFraction int    `koanf:"mutex-profile-fraction"`
	BlockProfileRate     int    `koanf:"block-profile-rate"`
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *Config) ApplyDefaults() {
	if c.Metrics.Interval == 0 {
		c.Metrics.Interval = DefaultMetricsInterval
	}
	if c.Tracing.SampleRatio == 0 {
		c.Tracing.SampleRatio = DefaultSampleRatio
	}
	if c.Profiling.Enabled {
		c.Profiling.applyDefaults()
	}
}

// Validate validates the observability configuration.
func (c *Config) Validate() error {
	if c.Tracing.SampleRatio < 0 || c.Tracing.SampleRatio > 1 {
		return fmt.Errorf("tracing sample-ratio must be between 0 and 1, got %f", c.Tracing.SampleRatio)
	}
	return nil
}

func (c *ProfilingConfig) applyDefaults() {
	if c.CPU == nil {
		c.CPU = new(true)
	}
	if c.Heap == nil {
		c.Heap = new(true)
	}
	if c.Goroutines == nil {
		c.Goroutines = new(true)
	}
	if c.Mutex == nil {
		c.Mutex = new(false)
	}
	if c.Block == nil {
		c.Block = new(false)
	}
	if *c.Mutex && c.MutexProfileFraction == 0 {
		c.MutexProfileFraction = DefaultMutexProfileFraction
	}
	if *c.Block && c.BlockProfileRate == 0 {
		c.BlockProfileRate = DefaultBlockProfileRate
	}
}
