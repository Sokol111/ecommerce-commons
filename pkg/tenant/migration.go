package tenant

import (
	"fmt"
	"net/url"

	"github.com/Sokol111/ecommerce-commons/pkg/mongo"
	"go.uber.org/zap"
)

// databaseMigrator applies migrations to a database at the given URI.
type databaseMigrator func(dbURL, migrationsPath string, log *zap.Logger) error

// MigrationRunner runs per-tenant database migrations.
type MigrationRunner struct {
	baseDatabase   string
	migrationsPath string
	baseURI        string
	migrate        databaseMigrator
	log            *zap.Logger
}

// NewMigrationRunner creates a MigrationRunner.
func NewMigrationRunner(cfg mongo.Config, log *zap.Logger) *MigrationRunner {
	baseCfg := cfg
	baseCfg.Database = ""
	return &MigrationRunner{
		baseDatabase:   cfg.Database,
		migrationsPath: cfg.Migrations.Path,
		baseURI:        baseCfg.BuildURI(),
		migrate:        mongo.MigrateDatabase,
		log:            log,
	}
}

// MigrateAll runs migrations for all given tenant slugs.
func (r *MigrationRunner) MigrateAll(slugs []string) error {
	if len(slugs) == 0 {
		r.log.Warn("No active tenants found, skipping migrations")
		return nil
	}

	r.log.Info("Running tenant migrations", zap.Int("tenants", len(slugs)))

	for _, slug := range slugs {
		if err := r.migrateTenant(slug); err != nil {
			return fmt.Errorf("migration failed for tenant %q: %w", slug, err)
		}
	}

	return nil
}

func (r *MigrationRunner) migrateTenant(slug string) error {
	database := fmt.Sprintf("%s_%s", r.baseDatabase, slug)

	u, err := url.Parse(r.baseURI)
	if err != nil {
		return fmt.Errorf("failed to parse base URI: %w", err)
	}
	u.Path = "/" + database

	r.log.Info("Running migration for tenant", zap.String("tenant", slug), zap.String("database", database))

	return r.migrate(u.String(), r.migrationsPath, r.log)
}
