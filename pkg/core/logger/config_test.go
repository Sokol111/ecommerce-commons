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
	assert.Equal(t, zapcore.ErrorLevel, cfg.StacktraceLevel)
	assert.False(t, cfg.Development)
}

func TestNewConfig_ValidConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		level               string
		stacktraceLevel     string
		development         bool
		expectedLevel       zapcore.Level
		expectedStacktrace  zapcore.Level
		expectedDevelopment bool
	}{
		{
			name:                "debug level with development mode",
			level:               "debug",
			stacktraceLevel:     "warn",
			development:         true,
			expectedLevel:       zapcore.DebugLevel,
			expectedStacktrace:  zapcore.WarnLevel,
			expectedDevelopment: true,
		},
		{
			name:                "info level production",
			level:               "info",
			stacktraceLevel:     "error",
			development:         false,
			expectedLevel:       zapcore.InfoLevel,
			expectedStacktrace:  zapcore.ErrorLevel,
			expectedDevelopment: false,
		},
		{
			name:                "warn level",
			level:               "warn",
			stacktraceLevel:     "panic",
			development:         false,
			expectedLevel:       zapcore.WarnLevel,
			expectedStacktrace:  zapcore.PanicLevel,
			expectedDevelopment: false,
		},
		{
			name:                "error level",
			level:               "error",
			stacktraceLevel:     "fatal",
			development:         true,
			expectedLevel:       zapcore.ErrorLevel,
			expectedStacktrace:  zapcore.FatalLevel,
			expectedDevelopment: true,
		},
		{
			name:                "fatal level",
			level:               "fatal",
			stacktraceLevel:     "fatal",
			development:         false,
			expectedLevel:       zapcore.FatalLevel,
			expectedStacktrace:  zapcore.FatalLevel,
			expectedDevelopment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: viper with specific logger configuration
			v := viper.New()
			v.Set("logger.level", tt.level)
			v.Set("logger.stacktraceLevel", tt.stacktraceLevel)
			v.Set("logger.development", tt.development)

			// When: creating config
			cfg, err := newConfig(v)

			// Then: configuration should match expected values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.Level)
			assert.Equal(t, tt.expectedStacktrace, cfg.StacktraceLevel)
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestNewConfig_InvalidLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		stacktrace  string
		expectedErr string
	}{
		{
			name:        "invalid log level",
			level:       "invalid",
			stacktrace:  "error",
			expectedErr: "invalid log level 'invalid'",
		},
		{
			name:        "invalid stacktrace level",
			level:       "info",
			stacktrace:  "invalid",
			expectedErr: "invalid stacktrace level 'invalid'",
		},
		{
			name:        "empty string is invalid",
			level:       "unknown",
			stacktrace:  "error",
			expectedErr: "invalid log level 'unknown'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: viper with invalid configuration
			v := viper.New()
			v.Set("logger.level", tt.level)
			v.Set("logger.stacktraceLevel", tt.stacktrace)

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
		expectedStacktrace  zapcore.Level
		expectedDevelopment bool
	}{
		{
			name: "only level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.level", "debug")
			},
			expectedLevel:       zapcore.DebugLevel,
			expectedStacktrace:  zapcore.DPanicLevel, // default
			expectedDevelopment: false,               // default
		},
		{
			name: "only development specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.development", true)
			},
			expectedLevel:       zapcore.InfoLevel,   // default
			expectedStacktrace:  zapcore.DPanicLevel, // default
			expectedDevelopment: true,
		},
		{
			name: "only stacktrace level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.stacktraceLevel", "warn")
			},
			expectedLevel:       zapcore.InfoLevel, // default
			expectedStacktrace:  zapcore.WarnLevel,
			expectedDevelopment: false, // default
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
			assert.Equal(t, tt.expectedStacktrace, cfg.StacktraceLevel)
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
