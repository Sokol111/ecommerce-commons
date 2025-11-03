package outbox

import (
	"context"
	"embed"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
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
			provideChannels,
			provideFetcher,
			provideSender,
			provideConfirmer,
			provideOutbox,
		),
		fx.Invoke(func(*fetcher, *sender, *confirmer) {}),
	)
}

func provideOutbox(lc fx.Lifecycle, log *zap.Logger, store Store, channels *channels, migrator migrations.Migrator, readiness health.Readiness) Outbox {
	if migrator != nil {
		readiness.AddOne()
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				defer readiness.Done()
				log.Info("running outbox migrations")
				if err := migrator.UpFromFS("outbox_migrations", migrationsFS, "migrations"); err != nil {
					return err
				}
				log.Info("outbox migrations completed")
				return nil
			},
		})
	} else {
		log.Warn("migrator not available, skipping outbox migrations")
	}

	return newOutbox(log, store, channels.entities)
}
