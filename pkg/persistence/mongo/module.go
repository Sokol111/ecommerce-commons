package mongo

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
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

func provideMongo(lc fx.Lifecycle, log *zap.Logger, conf Config, readiness health.ComponentManager) (Mongo, Admin, error) {
	m, err := newMongo(log, conf)

	if err != nil {
		return nil, nil, err
	}

	markReady := readiness.AddComponent("mongo-module")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			defer markReady()
			return m.connect(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return m.disconnect(ctx)
		},
	})

	return m, m, nil
}
