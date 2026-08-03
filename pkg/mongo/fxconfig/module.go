package fxconfig

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewMongoModule provides MongoDB components for dependency injection.
// By default, configuration is loaded from koanf.
func NewMongoModule() fx.Option {

	return fx.Options(
		fx.Provide(fx.Annotate(metricViews, fx.ResultTags(`group:"metric_views,flatten"`))),
		fx.Provide(
			provideMongoClient,
			provideDatabase,
			provideConfig,
			mongo.NewTxManager,
			fx.Annotate(
				mongo.NewSingleMigrationRunner,
				fx.As(new(mongo.MigrationRunner)),
			),
		),
		fx.Invoke(
			applyMongoLifecycle,
			registerMigrationHook,
		),
	)
}

func provideConfig(loader *config.Loader) (mongo.Config, error) {
	return config.Load[mongo.Config](loader, "mongo", nil)
}

func applyMongoLifecycle(lc fx.Lifecycle, log *zap.Logger, cfg mongo.Config, client *mongodriver.Client, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("mongo-module")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			c, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
			defer cancel()

			// Ping to establish actual connection (Connect was already called in newMongo)
			if err := client.Ping(c, nil); err != nil {
				return fmt.Errorf("failed to ping mongo: %w", err)
			}

			fields := []zap.Field{
				zap.String("database", cfg.Database),
				zap.Uint64("max-pool-size", cfg.MaxPoolSize),
				zap.Uint64("min-pool-size", cfg.MinPoolSize),
				zap.Duration("max-conn-idle-time", cfg.MaxConnIdleTime),
				zap.Duration("query-timeout", cfg.QueryTimeout),
			}
			if cfg.ConnectionString == "" {
				fields = append(fields,
					zap.String("host", cfg.Host),
					zap.Int("port", cfg.Port),
				)
			}

			log.Info("connected to mongo", fields...)

			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			c, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
			defer cancel()
			if err := client.Disconnect(c); err != nil {
				return fmt.Errorf("failed to disconnect from mongo: %w", err)
			}
			log.Info("disconnected from mongo")
			return nil
		},
	})
}

// RegisterMigrationHook registers a lifecycle hook to run database migrations on application start.
func registerMigrationHook(lc fx.Lifecycle, runner mongo.MigrationRunner, ready health.ComponentManager) {
	markReady := ready.AddComponent("migrations")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := runner.Run(ctx); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}
