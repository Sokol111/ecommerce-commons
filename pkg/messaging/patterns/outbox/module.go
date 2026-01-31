package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewOutboxModule() fx.Option {
	return fx.Module("outbox",
		fx.Decorate(
			func(log *zap.Logger) *zap.Logger {
				return log.With(zap.String("component", "outbox"))
			},
		),
		fx.Provide(
			newConfig,
			newOutboxRepository,
			newFetcher,
			newSender,
			newConfirmer,
			newOutbox,
			newTracePropagator,
			newMetadataPopulator,
			provideEntitiesChannel,
			provideDeliveryChannel,
		),
		fx.Invoke(
			worker.RunWorker[*fetcher]("outbox-fetcher", worker.WithTrafficReady()),
			worker.RunWorker[*sender]("outbox-sender", worker.WithTrafficReady()),
			worker.RunWorker[*confirmer]("outbox-confirmer", worker.WithTrafficReady()),
			ensureSchema,
		),
	)
}

func newMetadataPopulator(appCfg config.AppConfig) events.MetadataPopulator {
	return events.NewMetadataPopulator(appCfg.ServiceName)
}

func ensureSchema(lc fx.Lifecycle, log *zap.Logger, m mongo.Mongo, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("outbox-schema")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("ensuring outbox indexes")
			if err := EnsureIndexes(ctx, m); err != nil {
				return err
			}
			log.Info("outbox indexes ready")
			markReady()
			return nil
		},
	})
}

func provideEntitiesChannel() chan *outboxEntity {
	return make(chan *outboxEntity, 100)
}

func provideDeliveryChannel() chan kafka.Event {
	return make(chan kafka.Event, 1000)
}
