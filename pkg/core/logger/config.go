package logger

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

// Config holds the configuration for the logger.
type Config struct {
	// Level specifies the minimum logging level.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	Level zapcore.Level

	// Development enables development mode with console encoding and human-readable timestamps.
	// In production mode (false), JSON encoding is used.
	Development bool
}

func newConfig(v *viper.Viper) (Config, error) {
	sub := v.Sub("logger")
	if sub == nil {
		return Config{
			Level: zapcore.InfoLevel,
		}, nil
	}

	// Parse level from string first
	var rawCfg struct {
		Level       string `mapstructure:"level"`
		Development bool   `mapstructure:"development"`
	}

	if err := sub.UnmarshalExact(&rawCfg); err != nil {
		return Config{}, fmt.Errorf("failed to load logger config: %w", err)
	}

	// Parse level string to zapcore.Level
	level := zapcore.InfoLevel // default
	if rawCfg.Level != "" {
		parsedLevel, err := zapcore.ParseLevel(rawCfg.Level)
		if err != nil {
			return Config{}, fmt.Errorf("invalid log level '%s': %w", rawCfg.Level, err)
		}
		level = parsedLevel
	}

	cfg := Config{
		Level:       level,
		Development: rawCfg.Development,
	}

	return cfg, nil
}
