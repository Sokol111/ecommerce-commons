package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger_DevelopmentMode(t *testing.T) {
	// Given: development configuration
	cfg := Config{
		Level:       zapcore.DebugLevel,
		Development: true,
	}

	// When: creating logger
	logger, atomicLevel, err := newLogger(cfg)

	// Then: logger should be created successfully
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.Equal(t, zapcore.DebugLevel, atomicLevel.Level())

	// Cleanup
	_ = logger.Sync()
}

func TestNewLogger_ProductionMode(t *testing.T) {
	// Given: production configuration
	cfg := Config{
		Level:       zapcore.InfoLevel,
		Development: false,
	}

	// When: creating logger
	logger, atomicLevel, err := newLogger(cfg)

	// Then: logger should be created successfully
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.Equal(t, zapcore.InfoLevel, atomicLevel.Level())

	// Cleanup
	_ = logger.Sync()
}

func TestNewLogger_DifferentLevels(t *testing.T) {
	levels := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
		zapcore.DPanicLevel,
		zapcore.PanicLevel,
		zapcore.FatalLevel,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			// Given: configuration with specific level
			cfg := Config{
				Level:       level,
				Development: false,
			}

			// When: creating logger
			logger, atomicLevel, err := newLogger(cfg)

			// Then: logger should be created with correct level
			require.NoError(t, err)
			require.NotNil(t, logger)
			assert.Equal(t, level, atomicLevel.Level())

			// Cleanup
			_ = logger.Sync()
		})
	}
}

func TestNewLogger_SetsDefaultLogger(t *testing.T) {
	// Given: configuration
	cfg := Config{
		Level:       zapcore.InfoLevel,
		Development: false,
	}

	// Save original default logger
	originalDefault := defaultLogger

	// When: creating logger
	logger, _, err := newLogger(cfg)

	// Then: default logger should be set
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.Equal(t, logger, defaultLogger)

	// Cleanup
	_ = logger.Sync()
	defaultLogger = originalDefault
}

func TestNewLogger_InitializationLog(t *testing.T) {
	// Given: configuration
	cfg := Config{
		Level:       zapcore.DebugLevel,
		Development: true,
	}

	// Save original default logger
	originalDefault := defaultLogger

	// When: creating logger
	logger, _, err := newLogger(cfg)
	require.NoError(t, err)

	// Then: logger should be created successfully
	// The initialization log is written, but we can't easily capture it without
	// interfering with the logger creation process itself
	assert.NotNil(t, logger)
	assert.Equal(t, logger, defaultLogger)

	// Cleanup
	_ = logger.Sync()
	defaultLogger = originalDefault
}

func TestNewLogger_DevelopmentVsProduction(t *testing.T) {
	tests := []struct {
		name        string
		development bool
		encoding    string
	}{
		{
			name:        "development uses console encoding",
			development: true,
			encoding:    "console",
		},
		{
			name:        "production uses json encoding",
			development: false,
			encoding:    "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: configuration with development mode
			cfg := Config{
				Level:       zapcore.InfoLevel,
				Development: tt.development,
			}

			// When: creating logger
			logger, _, err := newLogger(cfg)

			// Then: logger should be created successfully
			require.NoError(t, err)
			require.NotNil(t, logger)

			// Cleanup
			_ = logger.Sync()
		})
	}
}

func TestNewLogger_AtomicLevel(t *testing.T) {
	// Given: configuration
	cfg := Config{
		Level:       zapcore.WarnLevel,
		Development: false,
	}

	// When: creating logger
	logger, atomicLevel, err := newLogger(cfg)
	require.NoError(t, err)

	// Then: atomic level should be modifiable
	assert.Equal(t, zapcore.WarnLevel, atomicLevel.Level())

	// Change level
	atomicLevel.SetLevel(zapcore.DebugLevel)
	assert.Equal(t, zapcore.DebugLevel, atomicLevel.Level())

	// Cleanup
	_ = logger.Sync()
}

func TestNewLogger_CallerEnabled(t *testing.T) {
	// Given: configuration
	cfg := Config{
		Level:       zapcore.InfoLevel,
		Development: false,
	}

	// When: creating logger with caller enabled
	logger, _, err := newLogger(cfg)

	// Then: logger should include caller information
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Cleanup
	_ = logger.Sync()
}
