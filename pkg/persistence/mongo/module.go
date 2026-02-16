package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// mongoOptions holds internal configuration for the Mongo module.
type mongoOptions struct {
	config *Config
}

// Option is a functional option for configuring the Mongo module.
type Option func(*mongoOptions)

// WithMongoConfig provides a static Config (useful for tests).
func WithMongoConfig(cfg Config) Option {
	return func(opts *mongoOptions) {
		opts.config = &cfg
	}
}

// NewMongoModule provides MongoDB components for dependency injection.
// By default, configuration is loaded from viper.
// Use WithMongoConfig for static config (useful for tests).
func NewMongoModule(opts ...Option) fx.Option {
	cfg := &mongoOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(
			provideMongo,
			provideConfig,
			newTxManager,
		),
	)
}

func provideConfig(opts *mongoOptions, v *viper.Viper) (Config, error) {
	var cfg Config
	if opts.config != nil {
		cfg = *opts.config
	} else if sub := v.Sub("mongo"); sub != nil {
		if err := sub.Unmarshal(&cfg); err != nil {
			return cfg, fmt.Errorf("failed to load mongo config: %w", err)
		}
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func provideMongo(lc fx.Lifecycle, log *zap.Logger, appConf config.AppConfig, conf Config, readiness health.ComponentManager, tp trace.TracerProvider, mp metric.MeterProvider) (Mongo, Admin, error) {
	m, err := newMongo(log, conf, appConf.ServiceName, tp, mp)

	if err != nil {
		return nil, nil, err
	}

	markReady := readiness.AddComponent("mongo-module")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := m.connect(ctx); err != nil {
				return err
			}

			// Run migrations after successful connection
			if err := runMigrations(conf, log); err != nil {
				return err
			}

			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return m.disconnect(ctx)
		},
	})

	return m, m, nil
}
