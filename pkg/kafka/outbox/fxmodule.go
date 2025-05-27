package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var OutboxModule = fx.Options(
	fx.Provide(
		provideNewOutbox,
	),
)

func provideNewOutbox(lc fx.Lifecycle, log *zap.Logger, producer producer.Producer, store Store) Outbox {
	o := NewOutbox(log, producer, store)

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
