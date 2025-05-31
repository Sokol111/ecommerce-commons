package mongo

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewMongoModule() fx.Option {
	return fx.Provide(
		provideMongo,
		newConfig,
		newTxManager,
	)
}

func provideMongo(lc fx.Lifecycle, log *zap.Logger, conf Config) (Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	m, err := newMongo(log, conf)

	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return m.Connect(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return m.Disconnect(ctx)
		},
	})

	return m, nil
}
