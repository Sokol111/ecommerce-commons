package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewOutboxModule() fx.Option {
	return fx.Provide(
		provideCollection,
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

func provideCollection(lc fx.Lifecycle, m mongo.Mongo) (*mongo.CollectionWrapper[collection], error) {
	wrapper := &mongo.CollectionWrapper[collection]{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Indexes are now managed via migrations (see db/migrations/000001_outbox_init.*). Just obtain the collection.
			wrapper.Coll = m.GetCollection("outbox")
			return nil
		},
	})
	return wrapper, nil
}
