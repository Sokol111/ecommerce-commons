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
	assert.Equal(t, zapcore.ErrorLevel, cfg.FxLevel)
	assert.False(t, cfg.Development)
}

func TestNewConfig_ValidConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		level               string
		stacktraceLevel     string
		fxLevel             string
		development         bool
		expectedLevel       zapcore.Level
		expectedStacktrace  zapcore.Level
		expectedFxLevel     zapcore.Level
		expectedDevelopment bool
	}{
		{
			name:                "debug level with development mode",
			level:               "debug",
			stacktraceLevel:     "warn",
			fxLevel:             "info",
			development:         true,
			expectedLevel:       zapcore.DebugLevel,
			expectedStacktrace:  zapcore.WarnLevel,
			expectedFxLevel:     zapcore.InfoLevel,
			expectedDevelopment: true,
		},
		{
			name:                "info level production",
			level:               "info",
			stacktraceLevel:     "error",
			fxLevel:             "warn",
			development:         false,
			expectedLevel:       zapcore.InfoLevel,
			expectedStacktrace:  zapcore.ErrorLevel,
			expectedFxLevel:     zapcore.WarnLevel,
			expectedDevelopment: false,
		},
		{
			name:                "warn level",
			level:               "warn",
			stacktraceLevel:     "panic",
			fxLevel:             "error",
			development:         false,
			expectedLevel:       zapcore.WarnLevel,
			expectedStacktrace:  zapcore.PanicLevel,
			expectedFxLevel:     zapcore.ErrorLevel,
			expectedDevelopment: false,
		},
		{
			name:                "error level",
			level:               "error",
			stacktraceLevel:     "fatal",
			fxLevel:             "debug",
			development:         true,
			expectedLevel:       zapcore.ErrorLevel,
			expectedStacktrace:  zapcore.FatalLevel,
			expectedFxLevel:     zapcore.DebugLevel,
			expectedDevelopment: true,
		},
		{
			name:                "fatal level",
			level:               "fatal",
			stacktraceLevel:     "fatal",
			fxLevel:             "warn",
			development:         false,
			expectedLevel:       zapcore.FatalLevel,
			expectedStacktrace:  zapcore.FatalLevel,
			expectedFxLevel:     zapcore.WarnLevel,
			expectedDevelopment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: viper with specific logger configuration
			v := viper.New()
			v.Set("logger.level", tt.level)
			v.Set("logger.stacktrace-level", tt.stacktraceLevel)
			v.Set("logger.fx-level", tt.fxLevel)
			v.Set("logger.development", tt.development)

			// When: creating config
			cfg, err := newConfig(v)

			// Then: configuration should match expected values
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, cfg.Level)
			assert.Equal(t, tt.expectedStacktrace, cfg.StacktraceLevel)
			assert.Equal(t, tt.expectedFxLevel, cfg.FxLevel)
			assert.Equal(t, tt.expectedDevelopment, cfg.Development)
		})
	}
}

func TestNewConfig_InvalidLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		stacktrace  string
		fxLevel     string
		expectedErr string
	}{
		{
			name:        "invalid log level",
			level:       "invalid",
			stacktrace:  "error",
			fxLevel:     "warn",
			expectedErr: "invalid log level 'invalid'",
		},
		{
			name:        "invalid stacktrace level",
			level:       "info",
			stacktrace:  "invalid",
			fxLevel:     "warn",
			expectedErr: "invalid stacktrace level 'invalid'",
		},
		{
			name:        "invalid fx level",
			level:       "info",
			stacktrace:  "error",
			fxLevel:     "invalid",
			expectedErr: "invalid fx level 'invalid'",
		},
		{
			name:        "empty string is invalid",
			level:       "unknown",
			stacktrace:  "error",
			fxLevel:     "warn",
			expectedErr: "invalid log level 'unknown'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: viper with invalid configuration
			v := viper.New()
			v.Set("logger.level", tt.level)
			v.Set("logger.stacktrace-level", tt.stacktrace)
			v.Set("logger.fx-level", tt.fxLevel)

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
		expectedFxLevel     zapcore.Level
		expectedDevelopment bool
	}{
		{
			name: "only level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.level", "debug")
			},
			expectedLevel:       zapcore.DebugLevel,
			expectedStacktrace:  zapcore.DPanicLevel, // default
			expectedFxLevel:     zapcore.ErrorLevel,  // default
			expectedDevelopment: false,               // default
		},
		{
			name: "only development specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.development", true)
			},
			expectedLevel:       zapcore.InfoLevel,   // default
			expectedStacktrace:  zapcore.DPanicLevel, // default
			expectedFxLevel:     zapcore.ErrorLevel,  // default
			expectedDevelopment: true,
		},
		{
			name: "only stacktrace level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.stacktrace-level", "warn")
			},
			expectedLevel:       zapcore.InfoLevel, // default
			expectedStacktrace:  zapcore.WarnLevel,
			expectedFxLevel:     zapcore.ErrorLevel, // default
			expectedDevelopment: false,              // default
		},
		{
			name: "only fx-level specified",
			setupViper: func(v *viper.Viper) {
				v.Set("logger.fx-level", "info")
			},
			expectedLevel:       zapcore.InfoLevel,   // default
			expectedStacktrace:  zapcore.DPanicLevel, // default
			expectedFxLevel:     zapcore.InfoLevel,
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
			assert.Equal(t, tt.expectedFxLevel, cfg.FxLevel)
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
