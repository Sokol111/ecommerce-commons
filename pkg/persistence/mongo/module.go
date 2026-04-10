package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/knadh/koanf/v2"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// mongoOptions holds internal configuration for the Mongo module.
type mongoOptions struct {
	config           *Config
	tenantMigrations bool
}

// Option is a functional option for configuring the Mongo module.
type Option func(*mongoOptions)

// WithMongoConfig provides a static Config (useful for tests).
func WithMongoConfig(cfg Config) Option {
	return func(opts *mongoOptions) {
		opts.config = &cfg
	}
}

// WithTenantMigrations enables per-tenant database migrations.
// When set, TenantSlugsProvider must be provided in the fx container.
// The provider fetches tenant slugs at startup, and migrations run
// against each tenant database ({database}_{slug}).
func WithTenantMigrations() Option {
	return func(opts *mongoOptions) {
		opts.tenantMigrations = true
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
		fx.Provide(
			provideMongo,
			provideConfig,
			newTxManager,
			fx.Annotate(MetricViews, fx.ResultTags(`group:"metric_views,flatten"`)),
		),
	)
}

func provideConfig(opts *mongoOptions, k *koanf.Koanf) (Config, error) {
	var cfg Config
	if opts.config != nil {
		cfg = *opts.config
	} else if k.Exists("mongo") {
		if err := k.Unmarshal("mongo", &cfg); err != nil {
			return cfg, fmt.Errorf("failed to load mongo config: %w", err)
		}
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

type provideMongoParams struct {
	fx.In

	Lifecycle      fx.Lifecycle
	Opts           *mongoOptions
	Log            *zap.Logger
	AppConf        config.AppConfig
	Conf           Config
	Readiness      health.ComponentManager
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	SlugsProvider  TenantSlugsProvider `optional:"true"`
}

func provideMongo(p provideMongoParams) (Mongo, Admin, error) {
	m, err := newMongo(p.Log, p.Conf, p.AppConf.ServiceName, p.TracerProvider, p.MeterProvider)

	if err != nil {
		return nil, nil, err
	}

	if p.Opts.tenantMigrations && p.SlugsProvider == nil {
		return nil, nil, fmt.Errorf("tenant migrations enabled but TenantSlugsProvider not provided in fx container")
	}

	markReady := p.Readiness.AddComponent("mongo-module")
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := m.connect(ctx); err != nil {
				return err
			}

			// Run migrations after successful connection
			if p.SlugsProvider != nil {
				if err := runTenantMigrations(ctx, p.Conf, p.SlugsProvider, p.Log); err != nil {
					return err
				}
			} else {
				if err := runMigrations(p.Conf, p.Log); err != nil {
					return err
				}
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
