package logger

import (
	"context"
	"fmt"
	"os"

	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

func CombineLogger(base *zap.Logger, ctx context.Context) *zap.Logger {
	if ctxLogger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return ctxLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, base.Core())
		}))
	}
	return base
}

func FromContext(ctx context.Context) *zap.Logger {
	if ctxLogger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return ctxLogger
	}
	return zap.L()
}

func NewZapLoggingModule() fx.Option {
	return fx.Options(
		fx.Provide(provideLogger),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)
}

func provideLogger(lc fx.Lifecycle, env config.Environment) (*zap.Logger, error) {
	logger, err := newLogger(env)

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

func newLogger(env config.Environment) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	switch env {
	case config.EnvProduction:
		logger, err = zap.NewProduction()
	case config.EnvDevelopment:
		logger, err = zap.NewDevelopment()
	default:
		logger = zap.NewExample()
	}

	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)

	return logger, nil
}
