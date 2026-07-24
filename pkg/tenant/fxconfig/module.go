package fxconfig

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
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
		fx.Provide(
			tenant.NewMongoRepository,
			tenant.NewMigrationRunner,
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

	if cfg.enableMigrations {
		modules = append(modules, fx.Invoke(registerMigrations))
	}

	return fx.Module("tenant-lifecycle", modules...)
}

// registerMigrations syncs the tenant registry and runs per-tenant migrations on startup.
func registerMigrations(lc fx.Lifecycle, syncer *tenant.TenantSyncer, runner *tenant.MigrationRunner, readiness health.ComponentManager) {
	markReady := readiness.AddComponent("tenant-migrations")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			slugs, err := syncer.Sync(ctx)
			if err != nil {
				return err
			}

			if err := runner.MigrateAll(slugs); err != nil {
				return err
			}

			markReady()
			return nil
		},
	})
}
