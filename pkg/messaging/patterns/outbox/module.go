package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewOutboxModule() fx.Option {
	return fx.Provide(
		newStore,
		provideOutbox,
	)
}

func provideOutbox(lc fx.Lifecycle, log *zap.Logger, producer producer.Producer, store Store) Outbox {
	o := newOutbox(log, producer, store)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			o.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			o.Stop(ctx)
			return nil
		},
	})
	return o
}
