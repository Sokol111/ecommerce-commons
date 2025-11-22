package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// defaultLogger holds the default logger instance created during initialization.
var defaultLogger *zap.Logger

func newLogger(conf Config) (*zap.Logger, zap.AtomicLevel, error) {
	var cfg zap.Config

	if conf.Development {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	// After Validate(), Level is guaranteed to be valid
	atomicLevel := zap.NewAtomicLevelAt(conf.Level)
	cfg.Level = atomicLevel

	// Use ISO8601 time encoding for consistency
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Build logger with optional features
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(conf.StacktraceLevel),
	}

	logger, err := cfg.Build(options...)
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}

	// Set as default logger for the package
	defaultLogger = logger

	logger.Info("logger initialized",
		zap.String("level", conf.Level.String()),
		zap.String("stacktrace_level", conf.StacktraceLevel.String()),
		zap.Bool("development", conf.Development),
		zap.Bool("caller_enabled", true),
		zap.String("encoding", map[bool]string{true: "console", false: "json"}[conf.Development]),
		zap.String("time_encoding", "ISO8601"),
	)

	return logger, atomicLevel, nil
}
