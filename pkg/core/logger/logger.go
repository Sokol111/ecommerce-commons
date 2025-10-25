package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

// LoggerCtxKey is the context key used to store and retrieve logger instances from context.
const LoggerCtxKey contextKey = "logger_ctx_key"

// FromContext extracts a logger from the context.
// If no logger is found in the context, it returns the global logger.
// This function is safe to call with a nil context.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	if ctxLogger, ok := ctx.Value(LoggerCtxKey).(*zap.Logger); ok && ctxLogger != nil {
		return ctxLogger
	}
	return zap.L()
}

// WithLogger returns a new context with the provided logger attached.
// This allows propagating logger instances through the application context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, LoggerCtxKey, logger)
}

func newLogger(conf Config) (*zap.Logger, error) {
	// Validate configuration before building logger
	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("logger configuration validation failed: %w", err)
	}

	var cfg zap.Config

	// Use development or production config based on configuration
	if conf.Development {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	// Set log level (defaults to info if empty)
	logLevel := conf.Level
	if logLevel == "" {
		logLevel = "info"
	}
	cfg.Level = zap.NewAtomicLevelAt(parseLogLevel(logLevel))

	// Use ISO8601 time encoding for consistency
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure output paths if provided
	if len(conf.OutputPaths) > 0 {
		cfg.OutputPaths = conf.OutputPaths
	}

	// Configure error output paths if provided
	if len(conf.ErrorOutputPaths) > 0 {
		cfg.ErrorOutputPaths = conf.ErrorOutputPaths
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)

	logger.Info("logger initialized",
		zap.String("level", logLevel),
		zap.Bool("development", conf.Development),
	)

	return logger, nil
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}
