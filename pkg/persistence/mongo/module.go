package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// mongoOptions holds internal configuration for the Mongo module.
type mongoOptions struct {
	config *Config
}

// MongoOption is a functional option for configuring the Mongo module.
type MongoOption func(*mongoOptions)

// WithMongoConfig provides a static Config (useful for tests).
func WithMongoConfig(cfg Config) MongoOption {
	return func(opts *mongoOptions) {
		opts.config = &cfg
	}
}

// NewMongoModule provides MongoDB components for dependency injection.
// By default, configuration is loaded from viper.
// Use WithMongoConfig for static config (useful for tests).
func NewMongoModule(opts ...MongoOption) fx.Option {
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

	applyDefaults(&cfg)

	return cfg, nil
}

func provideMongo(lc fx.Lifecycle, log *zap.Logger, appConf config.AppConfig, conf Config, readiness health.ComponentManager) (Mongo, Admin, error) {
	m, err := newMongo(log, conf, appConf.ServiceName)

	if err != nil {
		return nil, nil, err
	}

	markReady := readiness.AddComponent("mongo-module")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			defer markReady()
			return m.connect(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return m.disconnect(ctx)
		},
	})

	return m, m, nil
}
