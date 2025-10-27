package outbox

import (
	"context"
	"embed"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
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
	)
}

func provideOutbox(lc fx.Lifecycle, log *zap.Logger, producer producer.Producer, store Store, migrator migrations.Migrator, readiness health.Readiness) Outbox {
	// Run migrations when outbox is requested (lazy initialization)
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

	o := newOutbox(log, producer, store)

	readiness.AddOne()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			defer readiness.Done()
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
