package fxconfig

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ResolverInterceptorPriority puts tenant resolution before logger (18) so
// the tenant field is available in all subsequent logs.
const ResolverInterceptorPriority = 18

// ValidatorInterceptorPriority puts tenant claim validation after logger (26)
// so auth failures are logged with the resolved tenant field.
const ValidatorInterceptorPriority = 26

// tenantOptions holds internal configuration for the tenant module.
type tenantOptions struct {
	config *tenant.Config
}

// Option is a functional option for configuring the tenant module.
type Option func(*tenantOptions)

// WithTenantConfig provides a static Config (useful for tests).
func WithTenantConfig(cfg tenant.Config) Option {
	return func(opts *tenantOptions) {
		opts.config = &cfg
	}
}

// NewTenantModule provides tenant lifecycle management and Connect-RPC interceptors
// for dependency injection.
func NewTenantModule(opts ...Option) fx.Option {
	cfg := &tenantOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("tenant-lifecycle",
		fx.Supply(cfg),
		fx.Decorate(func(
			syncer *tenant.TenantSyncer,
			repo tenant.Repository,
			mongoConf mongo.Config,
			log *zap.Logger,
			cfg tenant.Config,
			runner mongo.MigrationRunner,
		) mongo.MigrationRunner {
			if !cfg.Enabled {
				return runner
			}
			return tenant.NewTenantMigrationRunner(syncer, repo, mongoConf, log)
		}),
		fx.Provide(provideConfig),
		fx.Provide(
			fx.Annotate(
				func(cfg tenant.Config) fx_interceptor.Interceptor {
					if !cfg.Enabled {
						return fx_interceptor.Interceptor{}
					}
					return fx_interceptor.Interceptor{Priority: ResolverInterceptorPriority, Handler: tenant.NewResolverInterceptor()}
				},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
			fx.Annotate(
				func(cfg tenant.Config) fx_interceptor.Interceptor {
					if !cfg.Enabled {
						return fx_interceptor.Interceptor{}
					}
					return fx_interceptor.Interceptor{Priority: ValidatorInterceptorPriority, Handler: tenant.NewValidatorInterceptor()}
				},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
			tenant.NewMongoRepository,
			tenant.NewTenantSyncer,
			tenant.NewLifecycle,
			fx.Annotate(
				tenant.NewMongoCleaner,
				fx.As(new(tenant.Cleaner)),
				fx.ResultTags(`group:"tenant_cleaners"`),
			),
			fx.Annotate(
				provideCleanupWorker,
				fx.ParamTags(``, ``, `group:"tenant_cleaners"`, ``),
			),
		),
		fx.Invoke(worker.RunWorker[tenant.CleanupWorker]("tenant-cleanup", worker.WithReady())),
	)
}

func provideCleanupWorker(cfg tenant.Config, repository tenant.Repository, cleaners []tenant.Cleaner, logger *zap.Logger) tenant.CleanupWorker {
	if !cfg.Enabled {
		return &tenant.NoopCleanupWorker{}
	}
	return tenant.NewDefaultCleanupWorker(repository, cleaners, logger)
}

func provideConfig(opts *tenantOptions, loader *config.Loader) (tenant.Config, error) {
	return config.Load[tenant.Config](loader, "multi-tenancy", opts.config)
}
