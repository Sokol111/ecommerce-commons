package logging

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/config"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

var ZapLoggingModule = fx.Options(
	fx.Provide(newLogger),
	fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: log}
	}),
)

func newLogger(lc fx.Lifecycle, env config.Environment) (*zap.Logger, error) {
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

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})

	return logger, nil
}
