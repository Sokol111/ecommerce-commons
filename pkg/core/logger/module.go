package logger

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// NewZapLoggingModule creates a new fx module for zap logger initialization.
// It provides a configured *zap.Logger instance and integrates with fx lifecycle.
func NewZapLoggingModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newConfig,
			provideLogger,
		),
		fx.Invoke(func(log *zap.Logger, conf Config) {
			log.Info("Logger initialized",
				zap.String("level", conf.Level.String()),
				zap.Bool("development", conf.Development),
				zap.String("stacktraceLevel", conf.StacktraceLevel.String()),
			)
		}),
		fx.WithLogger(func(log *zap.Logger, conf Config) fxevent.Logger {
			zapLogger := &fxevent.ZapLogger{Logger: log}

			// In development mode, show all fx internal events for debugging
			// In production, only show warnings and errors to reduce noise
			if conf.Development {
				zapLogger.UseLogLevel(zap.DebugLevel)
			} else {
				zapLogger.UseLogLevel(zap.WarnLevel)
			}

			return zapLogger
		}),
	)
}

func provideLogger(lc fx.Lifecycle, conf Config) (*zap.Logger, zap.AtomicLevel, error) {
	logger, atomicLevel, err := newLogger(conf)

	if err != nil {
		return nil, zap.AtomicLevel{}, fmt.Errorf("failed to create logger: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Best-effort sync, ignore errors.
			// Sync errors on stdout/stderr are expected on some systems.
			_ = logger.Sync()
			return nil
		},
	})

	return logger, atomicLevel, nil
}
