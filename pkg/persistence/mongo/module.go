package mongo

import (
	"context"

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
	config            *Config
	disableMigrations bool
}

// Option is a functional option for configuring the Mongo module.
type Option func(*mongoOptions)

// WithMongoConfig provides a static Config (useful for tests).
func WithMongoConfig(cfg Config) Option {
	return func(opts *mongoOptions) {
		opts.config = &cfg
	}
}

// WithoutMigrations disables automatic migrations on startup.
// Use when migrations are managed externally (e.g. by tenant.NewModule()).
func WithoutMigrations() Option {
	return func(opts *mongoOptions) {
		opts.disableMigrations = true
	}
}

// NewMongoModule provides MongoDB components for dependency injection.
// By default, configuration is loaded from koanf and migrations run on startup.
// Use WithMongoConfig for static config (useful for tests).
// Use WithoutMigrations when migrations are managed externally.
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

	modules := []fx.Option{
		fx.Supply(cfg),
		fx.Provide(providers...),
	}

	if !cfg.disableMigrations {
		modules = append(modules, fx.Invoke(registerMigrations))
	}

	return fx.Options(modules...)
}

func provideConfig(opts *mongoOptions, k *koanf.Koanf) (Config, error) {
	return config.Load[Config](k, "mongo", opts.config)
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
