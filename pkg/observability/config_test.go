package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T {
	return &v
}

func TestConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	t.Run("sets all defaults for empty config", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.ApplyDefaults()

		assert.Equal(t, DefaultMetricsInterval, cfg.Metrics.Interval)
		assert.Equal(t, DefaultSampleRatio, cfg.Tracing.SampleRatio)
	})

	t.Run("does not override explicitly set values", func(t *testing.T) {
		t.Parallel()
		cfg := Config{
			Metrics: MetricsConfig{
				Interval: 30 * time.Second,
			},
			Tracing: TracingConfig{
				SampleRatio: 0.5,
			},
		}
		cfg.ApplyDefaults()

		assert.Equal(t, 30*time.Second, cfg.Metrics.Interval)
		assert.Equal(t, 0.5, cfg.Tracing.SampleRatio)
	})

	t.Run("applies profiling defaults when enabled", func(t *testing.T) {
		t.Parallel()
		cfg := Config{
			Profiling: ProfilingConfig{
				Enabled: true,
			},
		}
		cfg.ApplyDefaults()

		assert.True(t, *cfg.Profiling.CPU)
		assert.True(t, *cfg.Profiling.Heap)
		assert.True(t, *cfg.Profiling.Goroutines)
		assert.False(t, *cfg.Profiling.Mutex)
		assert.False(t, *cfg.Profiling.Block)
	})

	t.Run("profiling defaults keep explicit nil when disabled", func(t *testing.T) {
		t.Parallel()
		cfg := Config{
			Profiling: ProfilingConfig{
				Enabled: false,
			},
		}
		cfg.ApplyDefaults()

		assert.Nil(t, cfg.Profiling.CPU)
		assert.Nil(t, cfg.Profiling.Heap)
	})

	t.Run("sets mutex and block default rates when enabled", func(t *testing.T) {
		t.Parallel()
		cfg := Config{
			Profiling: ProfilingConfig{
				Enabled: true,
				Mutex:   ptr(true),
				Block:   ptr(true),
			},
		}
		cfg.ApplyDefaults()

		assert.Equal(t, DefaultMutexProfileFraction, cfg.Profiling.MutexProfileFraction)
		assert.Equal(t, DefaultBlockProfileRate, cfg.Profiling.BlockProfileRate)
	})

	t.Run("keeps explicit mutex and block rates", func(t *testing.T) {
		t.Parallel()
		cfg := Config{
			Profiling: ProfilingConfig{
				Enabled:              true,
				Mutex:                ptr(true),
				Block:                ptr(true),
				MutexProfileFraction: 10,
				BlockProfileRate:     20,
			},
		}
		cfg.ApplyDefaults()

		assert.Equal(t, 10, cfg.Profiling.MutexProfileFraction)
		assert.Equal(t, 20, cfg.Profiling.BlockProfileRate)
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid sample ratios pass", func(t *testing.T) {
		t.Parallel()
		for _, ratio := range []float64{0, 0.5, 1} {
			cfg := Config{
				Tracing: TracingConfig{
					SampleRatio: ratio,
				},
			}
			err := cfg.Validate()
			assert.NoError(t, err)
		}
	})

	t.Run("invalid sample ratios fail", func(t *testing.T) {
		t.Parallel()
		for _, ratio := range []float64{-0.1, 1.1} {
			cfg := Config{
				Tracing: TracingConfig{
					SampleRatio: ratio,
				},
			}
			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "sample-ratio must be between 0 and 1")
		}
	})
}
