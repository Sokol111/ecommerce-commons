package fxconfig

import (
	"context"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"

	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/kafkaproto"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewOutboxModule provides outbox pattern components for reliable message delivery.
func NewOutboxModule() fx.Option {
	return fx.Module("outbox",
		fx.Decorate(
			func(log *zap.Logger) *zap.Logger {
				return log.With(zap.String("component", "outbox"))
			},
		),
		fx.Provide(
			provideConfig,
			outbox.NewOutboxRepository,
			outbox.NewFetcher,
			outbox.NewSender,
			outbox.NewConfirmer,
			outbox.NewTracePropagator,
			newHeaderPopulator,
			provideEntitiesChannel,
			provideConfirmChannel,
			fx.Private,
		),
		fx.Provide(outbox.NewOutbox),
		fx.Invoke(
			worker.RunWorker[*outbox.Fetcher]("outbox-fetcher", worker.WithTrafficReady()),
			worker.RunWorker[*outbox.Sender]("outbox-sender", worker.WithTrafficReady()),
			worker.RunWorker[*outbox.Confirmer]("outbox-confirmer", worker.WithTrafficReady()),
			ensureSchema,
		),
	)
}

func provideConfig(loader *coreconfig.Loader) (outbox.Config, error) {
	return coreconfig.Load[outbox.Config](loader, "outbox", nil)
}

func newHeaderPopulator(appCfg coreconfig.AppConfig) kafkaproto.HeaderPopulator {
	return kafkaproto.NewHeaderPopulator(appCfg.ServiceName)
}

func ensureSchema(lc fx.Lifecycle, log *zap.Logger, database *mongodriver.Database, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("outbox-schema")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("ensuring outbox indexes")
			if err := outbox.EnsureIndexes(ctx, database); err != nil {
				return err
			}
			log.Info("outbox indexes ready")
			markReady()
			return nil
		},
	})
}

func provideEntitiesChannel() chan *outbox.OutboxEntity {
	return make(chan *outbox.OutboxEntity, 100)
}

func provideConfirmChannel() chan outbox.ConfirmResult {
	return make(chan outbox.ConfirmResult, 1000)
}
