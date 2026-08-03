package fxconfig

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// loggerOptions holds internal configuration for the logger module.
type loggerOptions struct {
	config *logger.Config
}

// Option is a functional option for configuring the logger module.
type Option func(*loggerOptions)

// WithLoggerConfig provides a static Config (useful for tests).
func WithLoggerConfig(cfg logger.Config) Option {
	return func(opts *loggerOptions) {
		opts.config = &cfg
	}
}

// NewZapLoggingModule creates a new fx module for zap logger initialization.
// It provides a configured *zap.Logger instance and integrates with fx lifecycle.
// By default, loads from koanf configuration.
// Use WithLoggerConfig for static config (useful for tests).
func NewZapLoggingModule(opts ...Option) fx.Option {
	cfg := &loggerOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
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

func provideConfig(opts *loggerOptions, loader *config.Loader) (logger.Config, error) {
	return config.Load[logger.Config](loader, "logger", opts.config)
}
