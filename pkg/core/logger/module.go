package logger

import (
	"context"
	"fmt"
	"os"

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
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)
}

func provideLogger(lc fx.Lifecycle, conf Config) (*zap.Logger, error) {
	logger, err := newLogger(conf)

	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			err := logger.Sync()
			if err != nil {
				if pathErr, ok := err.(*os.PathError); ok && pathErr.Err.Error() == "invalid argument" {
					return nil
				}
				return err
			}
			return nil
		},
	})

	return logger, nil
}
