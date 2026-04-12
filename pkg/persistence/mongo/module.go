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

	providers := []any{
		provideMongo,
		provideConfig,
		newTxManager,
		fx.Annotate(MetricViews, fx.ResultTags(`group:"metric_views,flatten"`)),
	}

	if cfg.tenantMigrations {
		providers = append(providers, newTenantMigrationRunner)
	}

	modules := []fx.Option{
		fx.Supply(cfg),
		fx.Provide(providers...),
	}

	if cfg.tenantMigrations {
		modules = append(modules, fx.Invoke(registerTenantMigrations))
	} else {
		modules = append(modules, fx.Invoke(registerMigrations))
	}

	return fx.Options(modules...)
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
	Log            *zap.Logger
	AppConf        config.AppConfig
	Conf           Config
	Readiness      health.ComponentManager
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

func provideMongo(p provideMongoParams) (Mongo, Admin, error) {
	m, err := newMongo(p.Log, p.Conf, p.AppConf.ServiceName, p.TracerProvider, p.MeterProvider)

	if err != nil {
		return nil, nil, err
	}

	markReady := p.Readiness.AddComponent("mongo-module")
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := m.connect(ctx); err != nil {
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

// registerMigrations runs single-tenant migrations on startup.
func registerMigrations(lc fx.Lifecycle, cfg Config, log *zap.Logger, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("migrations")
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if err := runMigrations(cfg, log); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}

// registerTenantMigrations runs per-tenant migrations on startup.
// TenantSlugsProvider is a required (non-optional) dependency here, so fx will
// fail fast with a clear error if it's not provided.
func registerTenantMigrations(lc fx.Lifecycle, provider TenantSlugsProvider, cfg Config, log *zap.Logger, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("tenant-migrations")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := runTenantMigrations(ctx, cfg, provider, log); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}
