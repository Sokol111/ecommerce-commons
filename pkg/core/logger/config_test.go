package logger

import (
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	// Given: koanf without logger configuration
	k := koanf.New(".")

	// When: creating config
	cfg, err := newConfig(k)

	// Then: default values should be used
	require.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, cfg.Level)
	assert.False(t, cfg.Development)
}

func TestNewConfig_ValidConfiguration(t *testing.T) {
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

			// When: creating config
			cfg, err := newConfig(k)

			// Then: configuration should match expected values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.Level)
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestNewConfig_InvalidLevel(t *testing.T) {
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

			// When: creating config
			_, err := newConfig(k)

			// Then: should return error
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestNewConfig_PartialConfiguration(t *testing.T) {
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

			// When: creating config
			cfg, err := newConfig(k)

			// Then: should use defaults for unspecified values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.Level)
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestNewConfig_UnmarshalError(t *testing.T) {
	// Given: koanf with invalid type for boolean field
	k := koanf.New(".")
	k.Load(confmap.Provider(map[string]any{
		"logger.development": "not-a-boolean",
	}, "."), nil)

	// When: creating config
	_, err := newConfig(k)

	// Then: should return unmarshal error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load logger config")
}

func TestNewConfig_CaseSensitivity(t *testing.T) {
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

			// When: creating config
			cfg, err := newConfig(k)

			// Then: should handle case-insensitive parsing
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedLevel, cfg.Level)
			}
		})
	}
}
