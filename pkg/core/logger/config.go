package logger

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	// Level specifies the minimum logging level.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	Level zapcore.Level

	// Development enables development mode with console encoding and human-readable timestamps.
	// In production mode (false), JSON encoding is used.
	Development bool

	// StacktraceLevel sets the minimum level at which stacktraces are captured.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	// Defaults to ErrorLevel.
	StacktraceLevel zapcore.Level

	// FxLevel specifies the logging level for fx framework internal events.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	// Defaults to ErrorLevel to minimize noise from fx lifecycle events.
	FxLevel zapcore.Level
}

func newConfig(v *viper.Viper) (Config, error) {
	sub := v.Sub("logger")
	if sub == nil {
		return Config{
			Level:           zapcore.InfoLevel,
			StacktraceLevel: zapcore.ErrorLevel,
			FxLevel:         zapcore.ErrorLevel,
		}, nil
	}

	// Parse level from string first
	var rawCfg struct {
		Level           string `mapstructure:"level"`
		Development     bool   `mapstructure:"development"`
		StacktraceLevel string `mapstructure:"stacktrace-level"`
		FxLevel         string `mapstructure:"fx-level"`
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

	// Parse stacktrace level string to zapcore.Level
	stacktraceLevel := zapcore.DPanicLevel // default
	if rawCfg.StacktraceLevel != "" {
		parsedLevel, err := zapcore.ParseLevel(rawCfg.StacktraceLevel)
		if err != nil {
			return Config{}, fmt.Errorf("invalid stacktrace level '%s': %w", rawCfg.StacktraceLevel, err)
		}
		stacktraceLevel = parsedLevel
	}

	// Parse fx level string to zapcore.Level
	fxLevel := zapcore.ErrorLevel // default
	if rawCfg.FxLevel != "" {
		parsedLevel, err := zapcore.ParseLevel(rawCfg.FxLevel)
		if err != nil {
			return Config{}, fmt.Errorf("invalid fx level '%s': %w", rawCfg.FxLevel, err)
		}
		fxLevel = parsedLevel
	}

	cfg := Config{
		Level:           level,
		Development:     rawCfg.Development,
		StacktraceLevel: stacktraceLevel,
		FxLevel:         fxLevel,
	}

	return cfg, nil
}
