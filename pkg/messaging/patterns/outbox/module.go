package outbox

import (
	"context"
	"embed"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo/migrations"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed migrations/*.json
var migrationsFS embed.FS

func NewOutboxModule() fx.Option {
	return fx.Module("outbox",
		fx.Decorate(
			func(log *zap.Logger) *zap.Logger {
				return log.With(zap.String("component", "outbox"))
			},
		),
		fx.Provide(
			newStore,
			newFetcher,
			newSender,
			newConfirmer,
			newOutbox,
			provideEntitiesChannel,
			provideDeliveryChannel,
			worker.Register[*fetcher]("outbox-fetcher", worker.WithTrafficReady()),
			worker.Register[*sender]("outbox-sender", worker.WithTrafficReady()),
			worker.Register[*confirmer]("outbox-confirmer", worker.WithTrafficReady()),
		),
		fx.Invoke(func(*fetcher, *sender, *confirmer, *outbox) {}),
		fx.Invoke(runMigrations),
	)
}

func runMigrations(lc fx.Lifecycle, log *zap.Logger, migrator migrations.Migrator, readiness health.ComponentManager) {
	readiness.AddComponent("outbox-migrations")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("running outbox migrations")
			if err := migrator.UpFromFS("outbox_migrations", migrationsFS, "migrations"); err != nil {
				return err
			}
			log.Info("outbox migrations completed")
			readiness.MarkReady("outbox-migrations")
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
