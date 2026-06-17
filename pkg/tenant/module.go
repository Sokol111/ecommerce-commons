package tenant

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	"github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor"
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/fx"
)

// ResolverInterceptorPriority puts tenant resolution before logger (18) so
// the tenant field is available in all subsequent logs.
const ResolverInterceptorPriority = 18

// ValidatorInterceptorPriority puts tenant claim validation after logger (26)
// so auth failures are logged with the resolved tenant field.
const ValidatorInterceptorPriority = 26

// moduleOptions holds internal configuration for the tenant module.
type moduleOptions struct {
	enableMigrations bool
}

// Option is a functional option for configuring the tenant module.
type Option func(*moduleOptions)

// WithMigrations enables per-tenant database migrations on startup.
func WithMigrations() Option {
	return func(opts *moduleOptions) {
		opts.enableMigrations = true
	}
}

// NewModule provides tenant lifecycle management and Connect-RPC interceptors
// for dependency injection.
func NewModule(opts ...Option) fx.Option {
	cfg := &moduleOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	modules := []fx.Option{
		fx.Provide(
			newMongoRepository,
			newMigrationRunner,
			newTenantSyncer,
			newLifecycle,
			provideDatabaseResolver,
			fx.Annotate(
				newMongoCleaner,
				fx.As(new(Cleaner)),
				fx.ResultTags(`group:"tenant_cleaners"`),
			),
			fx.Annotate(
				newCleanupWorker,
				fx.ParamTags(``, `group:"tenant_cleaners"`, ``),
			),
			fx.Annotate(
				func() interceptor.Interceptor {
					return interceptor.Interceptor{
						Priority: ResolverInterceptorPriority,
						Handler:  NewResolverInterceptor(),
					}
				},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
			fx.Annotate(
				func() interceptor.Interceptor {
					return interceptor.Interceptor{
						Priority: ValidatorInterceptorPriority,
						Handler:  NewValidatorInterceptor(),
					}
				},
				fx.ResultTags(`group:"connect_interceptor"`),
			),
		),
		fx.Invoke(worker.RunWorker[*cleanupWorker]("tenant-cleanup", worker.WithReady())),
	}

	if cfg.enableMigrations {
		modules = append(modules, fx.Invoke(registerMigrations))
	}

	return fx.Module("tenant-lifecycle", modules...)
}

// registerMigrations syncs the tenant registry and runs per-tenant migrations on startup.
func registerMigrations(lc fx.Lifecycle, syncer *tenantSyncer, runner *migrationRunner, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("tenant-migrations")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			slugs, err := syncer.sync(ctx)
			if err != nil {
				return err
			}

			if err := runner.migrateAll(slugs); err != nil {
				return err
			}

			markReady()
			return nil
		},
	})
}

// provideDatabaseResolver provides a DatabaseResolver that extracts the tenant slug from context.
func provideDatabaseResolver() mongo.DatabaseResolver {
	return MustSlugFromContext
}
