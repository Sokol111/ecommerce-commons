package logger

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	// Given: viper without logger configuration
	v := viper.New()

	// When: creating config
	cfg, err := newConfig(v)

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
			// Given: viper with specific logger configuration
			v := viper.New()
			v.Set("logger.level", tt.level)
			v.Set("logger.development", tt.development)

			// When: creating config
			cfg, err := newConfig(v)

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
			// Given: viper with invalid configuration
			v := viper.New()
			v.Set("logger.level", tt.level)

			// When: creating config
			_, err := newConfig(v)

			// Then: should return error
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestNewConfig_PartialConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		setupViper          func(*viper.Viper)
		expectedLevel       zapcore.Level
		expectedDevelopment bool
	}{
		{
			name: "only level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.level", "debug")
			},
			expectedLevel:       zapcore.DebugLevel,
			expectedDevelopment: false, // default
		},
		{
			name: "only development specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.development", true)
			},
			expectedLevel:       zapcore.InfoLevel, // default
			expectedDevelopment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: viper with partial configuration
			v := viper.New()
			tt.setupViper(v)

			// When: creating config
			cfg, err := newConfig(v)

			// Then: should use defaults for unspecified values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.Level)
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestNewConfig_UnmarshalError(t *testing.T) {
	// Given: viper with invalid type for boolean field
	v := viper.New()
	v.Set("logger.development", "not-a-boolean")

	// When: creating config
	_, err := newConfig(v)

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
			// Given: viper with various case formats
			v := viper.New()
			v.Set("logger.level", tt.level)

			// When: creating config
			cfg, err := newConfig(v)

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
