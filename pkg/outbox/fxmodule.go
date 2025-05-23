package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewOutboxModule() fx.Option {
	return fx.Provide(
		ProvideNewOutbox,
	)
}

func ProvideNewOutbox(lc fx.Lifecycle, log *zap.Logger, producer kafka.Producer, store Store) Outbox {
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
