package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformEnvKey(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT", "observability.otel-collector-endpoint"},
		{"OBSERVABILITY__TRACING__ENABLED", "observability.tracing.enabled"},
		{"MONGO__CONNECTION_STRING", "mongo.connection-string"},
		{"MONGO__MAX_POOL_SIZE", "mongo.max-pool-size"},
		{"LOGGER__LEVEL", "logger.level"},
		{"KAFKA__SCHEMA_REGISTRY__AUTO_REGISTER_SCHEMAS", "kafka.schema-registry.auto-register-schemas"},
		{"APP_ENV", "app-env"},
		{"CONFIG_FILE", "config-file"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			key, _ := transformEnvKey(tt.env, "value")
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestNewKoanf_WithConfigFile(t *testing.T) {
	configContent := `
mongo:
  database: testdb
  max-pool-size: 10
observability:
  tracing:
    enabled: true
  metrics:
    enabled: false
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(configContent), 0o600))

	k, err := newKoanf(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "testdb", k.String("mongo.database"))
	assert.Equal(t, 10, k.Int("mongo.max-pool-size"))
	assert.True(t, k.Bool("observability.tracing.enabled"))
	assert.False(t, k.Bool("observability.metrics.enabled"))
}

func TestNewKoanf_EnvOverridesConfigFile(t *testing.T) {
	configContent := `
mongo:
  database: from-file
observability:
  otel-collector-endpoint: "from-file"
  tracing:
    enabled: false
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(configContent), 0o600))

	t.Setenv("MONGO__DATABASE", "from-env")
	t.Setenv("OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT", "localhost:4317")
	t.Setenv("OBSERVABILITY__TRACING__ENABLED", "true")

	k, err := newKoanf(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "from-env", k.String("mongo.database"))
	assert.Equal(t, "localhost:4317", k.String("observability.otel-collector-endpoint"))
	assert.True(t, k.Bool("observability.tracing.enabled"))
}

func TestNewKoanf_EnvOnlyWithoutPrefix(t *testing.T) {
	t.Setenv("OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT", "localhost:4317")

	k, err := newKoanf("")
	require.NoError(t, err)

	assert.Equal(t, "localhost:4317", k.String("observability.otel-collector-endpoint"))
}

func TestNewKoanf_NoConfigFile(t *testing.T) {
	k, err := newKoanf("")
	require.NoError(t, err)
	require.NotNil(t, k)
}

func TestNewKoanf_InvalidConfigFile(t *testing.T) {
	_, err := newKoanf("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestNewKoanf_UnmarshalWithPrefix(t *testing.T) {
	configContent := `
observability:
  tracing:
    enabled: true
    sample-ratio: 0.5
  metrics:
    enabled: true
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(configContent), 0o600))

	t.Setenv("OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT", "localhost:4317")

	k, err := newKoanf(tmpFile)
	require.NoError(t, err)

	type TracingConfig struct {
		Enabled     bool    `koanf:"enabled"`
		SampleRatio float64 `koanf:"sample-ratio"`
	}
	type MetricsConfig struct {
		Enabled bool `koanf:"enabled"`
	}
	type ObsConfig struct {
		OtelCollectorEndpoint string        `koanf:"otel-collector-endpoint"`
		Tracing               TracingConfig `koanf:"tracing"`
		Metrics               MetricsConfig `koanf:"metrics"`
	}

	var cfg ObsConfig
	require.NoError(t, k.Unmarshal("observability", &cfg))

	assert.Equal(t, "localhost:4317", cfg.OtelCollectorEndpoint)
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, 0.5, cfg.Tracing.SampleRatio)
	assert.True(t, cfg.Metrics.Enabled)
}
