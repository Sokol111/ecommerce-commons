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

// mongoOptions holds internal configuration for the Mongo module.
type mongoOptions struct {
	config *mongo.Config
}

// Option is a functional option for configuring the Mongo module.
type Option func(*mongoOptions)

// WithMongoConfig provides a static Config (useful for tests).
func WithMongoConfig(cfg mongo.Config) Option {
	return func(opts *mongoOptions) {
		opts.config = &cfg
	}
}

// NewMongoModule provides MongoDB components for dependency injection.
// By default, configuration is loaded from koanf.
// Use WithMongoConfig for static config (useful for tests).
func NewMongoModule(opts ...Option) fx.Option {
	cfg := &mongoOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(fx.Annotate(metricViews, fx.ResultTags(`group:"metric_views,flatten"`))),
		fx.Provide(
			provideMongoClient,
			provideDatabase,
			provideConfig,
			mongo.NewTxManager,
		),
		fx.Invoke(
			applyMongoLifecycle,
			registerMigrations,
		),
	)
}

func provideConfig(opts *mongoOptions, loader *config.Loader) (mongo.Config, error) {
	return config.Load[mongo.Config](loader, "mongo", opts.config)
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

// registerMigrations runs single-tenant migrations on startup.
func registerMigrations(lc fx.Lifecycle, cfg mongo.Config, log *zap.Logger, readiness health.ComponentManager) {
	if cfg.Migrations.Mode != "single" {
		log.Warn("skipping single mongo migrations", zap.String("mode", cfg.Migrations.Mode))
		return
	}
	markReady := readiness.AddComponent("single-migrations")
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if err := mongo.MigrateDatabase(cfg.BuildURI(), cfg.Migrations.Path, log); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}
