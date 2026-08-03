package fxconfig

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ResolverInterceptorPriority puts tenant resolution before logger (18) so
// the tenant field is available in all subsequent logs.
const ResolverInterceptorPriority = 18

// ValidatorInterceptorPriority puts tenant claim validation after logger (26)
// so auth failures are logged with the resolved tenant field.
const ValidatorInterceptorPriority = 26

// NewTenantModule provides tenant lifecycle management and Connect-RPC interceptors
// for dependency injection.
func NewTenantModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
		fx.Provide(fx.Annotate(
			provideTenantMigrationRunner,
			fx.ParamTags(``, `optional:"true"`, ``, ``, ``),
		)),
		fx.Decorate(decorateMigrationRunner),
		fx.Provide(
			fx.Annotate(
				provideResolverInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
			fx.Annotate(
				provideValidatorInterceptor,
				fx.ResultTags(`group:"connect_interceptor"`),
			),
			provideTenantRepository,
			provideTenantSyncer,
			fx.Annotate(
				provideTenantLifecycle,
				fx.ParamTags(``, ``, `optional:"true"`, ``),
			),
			provideTenantCleaner,
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

func provideConfig(loader *config.Loader) (tenant.Config, error) {
	return config.Load[tenant.Config](loader, "multi-tenancy", nil)
}

func provideTenantMigrationRunner(
	cfg tenant.Config,
	syncer *tenant.TenantSyncer,
	repo tenant.Repository,
	mongoConf mongo.Config,
	log *zap.Logger,
) *tenant.TenantMigrationRunner {
	if !cfg.Enabled {
		return nil
	}
	return tenant.NewTenantMigrationRunner(syncer, repo, mongoConf, log)
}

func decorateMigrationRunner(
	cfg tenant.Config,
	defaultRunner mongo.MigrationRunner,
	tenantRunner *tenant.TenantMigrationRunner,
) mongo.MigrationRunner {
	if !cfg.Enabled || tenantRunner == nil {
		return defaultRunner
	}
	return tenantRunner
}

func provideResolverInterceptor(cfg tenant.Config) fx_interceptor.Interceptor {
	if !cfg.Enabled {
		return fx_interceptor.Interceptor{}
	}
	return fx_interceptor.Interceptor{Priority: ResolverInterceptorPriority, Handler: tenant.NewResolverInterceptor()}
}

func provideValidatorInterceptor(cfg tenant.Config) fx_interceptor.Interceptor {
	if !cfg.Enabled {
		return fx_interceptor.Interceptor{}
	}
	return fx_interceptor.Interceptor{Priority: ValidatorInterceptorPriority, Handler: tenant.NewValidatorInterceptor()}
}

func provideTenantRepository(cfg tenant.Config, database *mongodriver.Database) tenant.Repository {
	if !cfg.Enabled {
		return nil
	}
	return tenant.NewMongoRepository(database)
}

func provideTenantSyncer(cfg tenant.Config, provider tenant.SlugsProvider, repo tenant.Repository, log *zap.Logger) *tenant.TenantSyncer {
	if !cfg.Enabled {
		return nil
	}
	return tenant.NewTenantSyncer(provider, repo, log)
}

func provideTenantLifecycle(cfg tenant.Config, repo tenant.Repository, runner *tenant.TenantMigrationRunner, log *zap.Logger) tenant.Lifecycle {
	if !cfg.Enabled {
		return nil
	}
	return tenant.NewLifecycle(repo, runner, log)
}

func provideTenantCleaner(cfg tenant.Config, database *mongodriver.Database, log *zap.Logger) tenant.Cleaner {
	if !cfg.Enabled {
		return nil
	}
	return tenant.NewMongoCleaner(database, log)
}
