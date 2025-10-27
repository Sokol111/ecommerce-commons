package migrations

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewMigrationsModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newConfig,
			provideMigrator,
		),
		fx.Invoke(func(m Migrator) {}),
	)
}

func provideMigrator(lc fx.Lifecycle, log *zap.Logger, conf Config, m mongo.Mongo) (Migrator, error) {
	migrator, err := newMigrator(m.GetDatabase(), log, conf.LockingTimeout)
	if err != nil {
		return nil, err
	}

	if conf.AutoMigrate {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				log.Info("auto-running migrations on startup",
					zap.String("collection", conf.CollectionName),
					zap.String("path", conf.MigrationsPath),
					zap.Duration("locking-timeout", conf.GetLockingTimeoutDuration()))

				if err := migrator.Up(conf.CollectionName, conf.MigrationsPath); err != nil {
					return fmt.Errorf("failed to run migrations: %w", err)
				}

				return nil
			},
		})
		log.Info("migrations auto-run is enabled, migrator also available in DI",
			zap.Duration("locking-timeout", conf.GetLockingTimeoutDuration()))
	} else {
		log.Info("migrations auto-run is disabled, migrator available for manual use")
	}

	return migrator, nil
}
