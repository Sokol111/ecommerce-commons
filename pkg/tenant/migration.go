package tenant

import (
	"fmt"
	"net/url"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/zap"
)

// databaseMigrator applies migrations to a database at the given URI.
type databaseMigrator func(dbURL, migrationsPath string, log *zap.Logger) error

// migrationRunner runs per-tenant database migrations.
type migrationRunner struct {
	baseDatabase   string
	migrationsPath string
	baseURI        string
	migrate        databaseMigrator
	log            *zap.Logger
}

// newMigrationRunner creates a migrationRunner.
func newMigrationRunner(cfg mongo.Config, log *zap.Logger) *migrationRunner {
	baseCfg := cfg
	baseCfg.Database = ""
	return &migrationRunner{
		baseDatabase:   cfg.Database,
		migrationsPath: cfg.Migrations.Path,
		baseURI:        baseCfg.BuildURI(),
		migrate:        mongo.MigrateDatabase,
		log:            log,
	}
}

// migrateAll runs migrations for all given tenant slugs.
func (r *migrationRunner) migrateAll(slugs []string) error {
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

func (r *migrationRunner) migrateTenant(slug string) error {
	database := fmt.Sprintf("%s_%s", r.baseDatabase, slug)

	u, err := url.Parse(r.baseURI)
	if err != nil {
		return fmt.Errorf("failed to parse base URI: %w", err)
	}
	u.Path = "/" + database

	r.log.Info("Running migration for tenant", zap.String("tenant", slug), zap.String("database", database))

	return r.migrate(u.String(), r.migrationsPath, r.log)
}
