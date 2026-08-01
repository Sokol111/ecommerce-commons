package tenant

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"go.uber.org/zap"
)

// TenantMigrationRunner implements mongo.MigrationRunner for multi-tenant services.
// When Run() is called: first sync the registry, then migrate each tenant.
type TenantMigrationRunner struct {
	syncer         *TenantSyncer
	repo           Repository
	baseURI        string
	baseDatabase   string
	migrationsPath string
	log            *zap.Logger
}

// NewTenantMigrationRunner creates TenantMigrationRunner.
func NewTenantMigrationRunner(
	syncer *TenantSyncer,
	repo Repository,
	cfg mongo.Config,
	log *zap.Logger,
) *TenantMigrationRunner {
	return &TenantMigrationRunner{
		syncer:         syncer,
		repo:           repo,
		baseURI:        cfg.BuildBaseURI(),
		baseDatabase:   cfg.Database,
		migrationsPath: cfg.Migrations.Path,
		log:            log,
	}
}

// Run synchronizes the tenant registry and migrates all tenant databases.
func (r *TenantMigrationRunner) Run(ctx context.Context) error {
	// 1. Sync tenant registry
	if _, err := r.syncer.Sync(ctx); err != nil {
		return fmt.Errorf("tenant sync failed: %w", err)
	}

	// 2. Read active tenants
	records, err := r.repo.FindActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active tenants: %w", err)
	}
	if len(records) == 0 {
		r.log.Warn("No active tenants found, skipping migrations")
		return nil
	}

	// 3. Migrate each tenant database
	r.log.Info("Running tenant migrations", zap.Int("tenants", len(records)))
	for _, rec := range records {
		if err := r.MigrateTenant(rec.Slug); err != nil {
			return fmt.Errorf("migration failed for tenant %q: %w", rec.Slug, err)
		}
	}
	return nil
}

// MigrateTenant migrates the database for a single tenant identified by its slug.
func (r *TenantMigrationRunner) MigrateTenant(slug string) error {
	database := fmt.Sprintf("%s_%s", r.baseDatabase, slug)

	u, err := url.Parse(r.baseURI)
	if err != nil {
		return fmt.Errorf("failed to parse base mongo uri: %w", err)
	}
	u.Path = "/" + database
	dbURL := u.String()

	r.log.Info("Running migration for tenant",
		zap.String("tenant", slug),
		zap.String("database", database),
	)

	return mongo.MigrateDatabase(dbURL, r.migrationsPath, r.log)
}
