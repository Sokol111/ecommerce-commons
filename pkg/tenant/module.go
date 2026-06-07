package tenant

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	httpmw "github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewModule provides tenant lifecycle management for dependency injection.
// It expects the following to be provided in the fx container:
//   - tenant.SlugsProvider (e.g. from tenantapi.NewTenantSlugsModule())
//   - *mongodriver.Database (from persistence/mongo module)
//   - mongo.Config (from persistence/mongo module)
func NewModule() fx.Option {
	return fx.Module("tenant-lifecycle",
		fx.Provide(
			newMongoRepository,
			newMigrationRunner,
			newTenantSyncer,
			newLifecycle,
			fx.Annotate(
				newMongoCleanupCleaner,
				fx.As(new(Cleaner)),
				fx.ResultTags(`group:"tenant_cleaners"`),
			),
			fx.Annotate(
				newCleanupWorker,
				fx.ParamTags(``, `group:"tenant_cleaners"`, ``),
			),
			fx.Annotate(
				func(log *zap.Logger) httpmw.Middleware {
					return httpmw.Middleware{
						Priority: 25,
						Handler:  Middleware(log),
					}
				},
				fx.ResultTags(`group:"ogen_mw"`),
			),
		),
		fx.Invoke(registerMigrations),
		fx.Invoke(worker.RunWorker[*cleanupWorker]("tenant-cleanup", worker.WithReady())),
	)
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
