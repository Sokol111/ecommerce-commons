package logger

import (
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestConfig_DefaultValues(t *testing.T) {
	// Given: koanf without logger configuration
	k := koanf.New(".")

	// When: loading config via generic loader
	cfg, err := config.Load[Config](k, "logger", nil)

	// Then: default values should be used
	require.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, cfg.ParsedLevel())
	assert.False(t, cfg.Development)
}

func TestConfig_ValidConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		level               string
		development         bool
		expectedLevel       zapcore.Level
		expectedDevelopment bool
	}{
		{
			name:                "debug level with development mode",
			level:               "debug",
			development:         true,
			expectedLevel:       zapcore.DebugLevel,
			expectedDevelopment: true,
		},
		{
			name:                "info level production",
			level:               "info",
			development:         false,
			expectedLevel:       zapcore.InfoLevel,
			expectedDevelopment: false,
		},
		{
			name:                "warn level",
			level:               "warn",
			development:         false,
			expectedLevel:       zapcore.WarnLevel,
			expectedDevelopment: false,
		},
		{
			name:                "error level",
			level:               "error",
			development:         true,
			expectedLevel:       zapcore.ErrorLevel,
			expectedDevelopment: true,
		},
		{
			name:                "fatal level",
			level:               "fatal",
			development:         false,
			expectedLevel:       zapcore.FatalLevel,
			expectedDevelopment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: koanf with specific logger configuration
			k := koanf.New(".")
			k.Load(confmap.Provider(map[string]any{
				"logger.level":       tt.level,
				"logger.development": tt.development,
			}, "."), nil)

			// When: loading config via generic loader
			cfg, err := config.Load[Config](k, "logger", nil)

			// Then: configuration should match expected values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.ParsedLevel())
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestConfig_InvalidLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectedErr string
	}{
		{
			name:        "invalid log level",
			level:       "invalid",
			expectedErr: "invalid log level 'invalid'",
		},
		{
			name:        "unknown level",
			level:       "unknown",
			expectedErr: "invalid log level 'unknown'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: koanf with invalid configuration
			k := koanf.New(".")
			k.Load(confmap.Provider(map[string]any{
				"logger.level": tt.level,
			}, "."), nil)

			// When: loading config via generic loader
			_, err := config.Load[Config](k, "logger", nil)

			// Then: should return error
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestConfig_PartialConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		setupKoanf          func(*koanf.Koanf)
		expectedLevel       zapcore.Level
		expectedDevelopment bool
	}{
		{
			name: "only level specified",
			setupKoanf: func(k *koanf.Koanf) {
				k.Load(confmap.Provider(map[string]any{
					"logger.level": "debug",
				}, "."), nil)
			},
			expectedLevel:       zapcore.DebugLevel,
			expectedDevelopment: false, // default
		},
		{
			name: "only development specified",
			setupKoanf: func(k *koanf.Koanf) {
				k.Load(confmap.Provider(map[string]any{
					"logger.development": true,
				}, "."), nil)
			},
			expectedLevel:       zapcore.InfoLevel, // default
			expectedDevelopment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: koanf with partial configuration
			k := koanf.New(".")
			tt.setupKoanf(k)

			// When: loading config via generic loader
			cfg, err := config.Load[Config](k, "logger", nil)

			// Then: should use defaults for unspecified values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.ParsedLevel())
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestConfig_UnmarshalError(t *testing.T) {
	// Given: koanf with invalid type for boolean field
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]any{
		"logger.development": "not-a-boolean",
	}, "."), nil)

	// When: loading config via generic loader
	_, err := config.Load[Config](k, "logger", nil)

	// Then: should return unmarshal error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load logger config")
}

func TestConfig_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zapcore.Level
		shouldError   bool
	}{
		{
			name:          "lowercase debug",
			level:         "debug",
			expectedLevel: zapcore.DebugLevel,
			shouldError:   false,
		},
		{
			name:          "uppercase DEBUG",
			level:         "DEBUG",
			expectedLevel: zapcore.DebugLevel,
			shouldError:   false,
		},
		{
			name:          "mixed case DeBuG",
			level:         "DeBuG",
			expectedLevel: zapcore.DebugLevel,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: koanf with various case formats
			k := koanf.New(".")
			k.Load(confmap.Provider(map[string]any{
				"logger.level": tt.level,
			}, "."), nil)

			// When: loading config via generic loader
			cfg, err := config.Load[Config](k, "logger", nil)

			// Then: should handle case-insensitive parsing
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedLevel, cfg.ParsedLevel())
			}
		})
	}
}

func TestConfig_WithOverride(t *testing.T) {
	// Given: koanf with some config (should be ignored when override is provided)
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]any{
		"logger.level": "error",
	}, "."), nil)

	// When: loading config with override
	override := &Config{Level: "debug", Development: true}
	cfg, err := config.Load[Config](k, "logger", override)

	// Then: override values should be used
	require.NoError(t, err)
	assert.Equal(t, zapcore.DebugLevel, cfg.ParsedLevel())
	assert.True(t, cfg.Development)
}
