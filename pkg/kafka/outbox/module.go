package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
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
			wrapper.Coll = m.GetCollection("outbox")
			err := m.CreateIndexes(ctx, "outbox", []mongodriver.IndexModel{
				{Keys: bson.D{{Key: "createdAt", Value: 1}}},
				{Keys: bson.D{{Key: "lockExpiresAt", Value: 1}}},
			})
			if err != nil {
				return err
			}
			return nil
		},
	})
	return wrapper, nil
}
