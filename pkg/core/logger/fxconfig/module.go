package fxconfig

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// NewZapLoggingModule creates a new fx module for zap logger initialization.
// It provides a configured *zap.Logger instance and integrates with fx lifecycle.
// By default, loads from koanf configuration.
func NewZapLoggingModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
		fx.Provide(logger.NewLogger),
		fx.Invoke(func(log *zap.Logger, conf logger.Config) {
			log.Info("Logger initialized",
				zap.String("level", conf.ParsedLevel().String()),
				zap.Bool("development", conf.Development),
			)
		}),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			zapLogger := &fxevent.ZapLogger{Logger: log}
			// Use DebugLevel so fx events are hidden when logger level is info or higher
			zapLogger.UseLogLevel(zap.DebugLevel)
			return zapLogger
		}),
		fx.Invoke(func(lc fx.Lifecycle, log *zap.Logger) {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					// Best-effort sync, ignore errors.
					// Sync errors on stdout/stderr are expected on some systems.
					_ = log.Sync() //nolint:errcheck // best-effort sync
					return nil
				},
			})
		}),
	)
}

func provideConfig(loader *config.Loader) (logger.Config, error) {
	return config.Load[logger.Config](loader, "logger", nil)
}
