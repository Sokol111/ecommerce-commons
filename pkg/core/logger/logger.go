package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

// LoggerCtxKey is the context key used to store and retrieve logger instances from context.
var loggerCtxKey = contextKey{}

// FromContext extracts a logger from the context.
// If no logger is found in the context, it returns the global logger.
// This function is safe to call with a nil context.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	if ctxLogger, ok := ctx.Value(loggerCtxKey).(*zap.Logger); ok && ctxLogger != nil {
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
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func newLogger(conf Config) (*zap.Logger, zap.AtomicLevel, error) {
	if err := conf.Validate(); err != nil {
		return nil, zap.AtomicLevel{}, fmt.Errorf("logger configuration validation failed: %w", err)
	}

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

	// Configure output paths if provided
	if len(conf.OutputPaths) > 0 {
		cfg.OutputPaths = conf.OutputPaths
	}

	// Configure error output paths if provided
	if len(conf.ErrorOutputPaths) > 0 {
		cfg.ErrorOutputPaths = conf.ErrorOutputPaths
	}

	// Build logger with optional features
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(conf.StacktraceLevel),
	}

	logger, err := cfg.Build(options...)
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}

	zap.ReplaceGlobals(logger)

	logger.Info("logger initialized",
		zap.String("level", conf.Level.String()),
		zap.Bool("development", conf.Development),
	)

	return logger, atomicLevel, nil
}
