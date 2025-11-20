package logger

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	// Level specifies the minimum logging level.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	Level zapcore.Level `mapstructure:"level"`

	// Development enables development mode with console encoding and human-readable timestamps.
	// In production mode (false), JSON encoding is used.
	Development bool `mapstructure:"development"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// If empty, defaults to stderr.
	OutputPaths []string `mapstructure:"outputPaths"`

	// ErrorOutputPaths is a list of URLs or file paths to write internal logger errors to.
	// If empty, defaults to stderr.
	ErrorOutputPaths []string `mapstructure:"errorOutputPaths"`

	// StacktraceLevel sets the minimum level at which stacktraces are captured.
	// Use zapcore constants: DebugLevel, InfoLevel, WarnLevel, ErrorLevel, DPanicLevel, PanicLevel, FatalLevel
	// Defaults to ErrorLevel.
	StacktraceLevel zapcore.Level `mapstructure:"stacktraceLevel"`
}

func (c Config) Validate() error {
	if err := validatePaths(c.OutputPaths, "outputPaths"); err != nil {
		return err
	}

	if err := validatePaths(c.ErrorOutputPaths, "errorOutputPaths"); err != nil {
		return err
	}

	return nil
}

func validatePaths(paths []string, fieldName string) error {
	for i, path := range paths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("%s[%d] cannot be empty or whitespace", fieldName, i)
		}
	}
	return nil
}

func newConfig(v *viper.Viper) (Config, error) {
	sub := v.Sub("logger")
	if sub == nil {
		return Config{
			Level:           zapcore.InfoLevel,
			StacktraceLevel: zapcore.ErrorLevel,
		}, nil
	}

	// Parse level from string first
	var rawCfg struct {
		Level            string   `mapstructure:"level"`
		Development      bool     `mapstructure:"development"`
		OutputPaths      []string `mapstructure:"outputPaths"`
		ErrorOutputPaths []string `mapstructure:"errorOutputPaths"`
		StacktraceLevel  string   `mapstructure:"stacktraceLevel"`
	}

	if err := sub.Unmarshal(&rawCfg); err != nil {
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
	stacktraceLevel := zapcore.ErrorLevel // default
	if rawCfg.StacktraceLevel != "" {
		parsedLevel, err := zapcore.ParseLevel(rawCfg.StacktraceLevel)
		if err != nil {
			return Config{}, fmt.Errorf("invalid stacktrace level '%s': %w", rawCfg.StacktraceLevel, err)
		}
		stacktraceLevel = parsedLevel
	}

	cfg := Config{
		Level:            level,
		Development:      rawCfg.Development,
		OutputPaths:      rawCfg.OutputPaths,
		ErrorOutputPaths: rawCfg.ErrorOutputPaths,
		StacktraceLevel:  stacktraceLevel,
	}

	return cfg, nil
}
