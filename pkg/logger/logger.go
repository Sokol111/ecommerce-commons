package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var CtxKey ctxKey = ctxKey{}

func FromContext(ctx context.Context) *zap.Logger {
	if ctxLogger, ok := ctx.Value(CtxKey).(*zap.Logger); ok {
		return ctxLogger
	}
	return zap.L()
}

func newLogger(logLevel string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	cfg.Level = zap.NewAtomicLevelAt(parseLogLevel(logLevel))
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Build()

	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)

	logger.Info("logger initialized", zap.String("level", logLevel))

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
	default:
		return zapcore.InfoLevel
	}
}
