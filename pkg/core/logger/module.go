package logger

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// loggerOptions holds internal configuration for the logger module.
type loggerOptions struct {
	config *Config
}

// LoggerOption is a functional option for configuring the logger module.
type LoggerOption func(*loggerOptions)

// WithLoggerConfig provides a static Config (useful for tests).
func WithLoggerConfig(cfg Config) LoggerOption {
	return func(opts *loggerOptions) {
		opts.config = &cfg
	}
}

// NewZapLoggingModule creates a new fx module for zap logger initialization.
// It provides a configured *zap.Logger instance and integrates with fx lifecycle.
// By default, loads from viper configuration.
// Use WithLoggerConfig for static config (useful for tests).
func NewZapLoggingModule(opts ...LoggerOption) fx.Option {
	cfg := &loggerOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Provide(func(v *viper.Viper) (Config, error) {
			if cfg.config != nil {
				return *cfg.config, nil
			}
			return newConfig(v)
		}),
		fx.Provide(provideLogger),
		fx.Invoke(func(log *zap.Logger, conf Config) {
			log.Info("Logger initialized",
				zap.String("level", conf.Level.String()),
				zap.Bool("development", conf.Development),
			)
		}),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			zapLogger := &fxevent.ZapLogger{Logger: log}
			// Use DebugLevel so fx events are hidden when logger level is info or higher
			zapLogger.UseLogLevel(zap.DebugLevel)
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
			_ = logger.Sync() //nolint:errcheck // best-effort sync
			return nil
		},
	})

	return logger, atomicLevel, nil
}
