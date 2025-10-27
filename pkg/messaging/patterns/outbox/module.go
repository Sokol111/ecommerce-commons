package outbox

import (
	"context"
	"embed"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo/migrations"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed migrations/*.json
var migrationsFS embed.FS

func NewOutboxModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newStore,
			provideOutbox,
		),
		fx.Invoke(runMigrations),
	)
}

func runMigrations(lc fx.Lifecycle, log *zap.Logger, migrator migrations.Migrator) {
	if migrator == nil {
		log.Warn("migrator not available, skipping outbox migrations")
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("running outbox migrations")
			if err := migrator.UpFromFS("outbox_migrations", migrationsFS, "migrations"); err != nil {
				return err
			}
			log.Info("outbox migrations completed")
			return nil
		},
	})
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
