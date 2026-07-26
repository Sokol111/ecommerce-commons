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

// NewModule provides tenant lifecycle management and Connect-RPC interceptors
// for dependency injection.
func NewModule(opts ...Option) fx.Option {
	cfg := &tenantOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	modules := []fx.Option{
		fx.Supply(cfg),
		fx.Supply(
			fx.Annotate(
				fx_interceptor.Interceptor{Priority: ResolverInterceptorPriority, Handler: tenant.NewResolverInterceptor()},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Supply(
			fx.Annotate(
				fx_interceptor.Interceptor{Priority: ValidatorInterceptorPriority, Handler: tenant.NewValidatorInterceptor()},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Decorate(func(
			syncer *tenant.TenantSyncer,
			repo tenant.Repository,
			cfg mongo.Config,
			log *zap.Logger,
		) mongo.MigrationRunner {
			return tenant.NewTenantMigrationRunner(syncer, repo, cfg, log)
		}),
		fx.Provide(
			provideConfig,
			tenant.NewMongoRepository,
			tenant.NewTenantSyncer,
			tenant.NewLifecycle,
			fx.Annotate(
				tenant.NewMongoCleaner,
				fx.As(new(tenant.Cleaner)),
				fx.ResultTags(`group:"tenant_cleaners"`),
			),
			fx.Annotate(
				tenant.NewCleanupWorker,
				fx.ParamTags(``, `group:"tenant_cleaners"`, ``),
			),
		),
		fx.Invoke(worker.RunWorker[*tenant.CleanupWorker]("tenant-cleanup", worker.WithReady())),
	}

	return fx.Module("tenant-lifecycle", modules...)
}

func provideConfig(opts *tenantOptions, loader *config.Loader) (tenant.Config, error) {
	return config.Load[tenant.Config](loader, "multi-tenancy", opts.config)
}
