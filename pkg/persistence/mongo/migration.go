package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb" // MongoDB driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // File source
	"go.uber.org/zap"
)

// TenantSlugsProvider fetches the list of active tenant slugs at startup.
// Implementations typically call tenant-service API.
type TenantSlugsProvider interface {
	GetSlugs(ctx context.Context) ([]string, error)
}

// runMigrations applies database migrations if enabled in config.
func runMigrations(cfg Config, log *zap.Logger) error {
	if cfg.Migrations.Disabled {
		log.Info("Database migrations disabled")
		return nil
	}

	return migrateDatabase(cfg.BuildURI(), cfg.Migrations.Path, log)
}

// runTenantMigrations fetches tenant slugs and runs migrations for each tenant database.
func runTenantMigrations(ctx context.Context, cfg Config, provider TenantSlugsProvider, log *zap.Logger) error {
	if cfg.Migrations.Disabled {
		log.Info("Database migrations disabled")
		return nil
	}

	slugs, err := provider.GetSlugs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant slugs: %w", err)
	}

	log.Info("Running tenant migrations", zap.Int("tenants", len(slugs)))

	for _, slug := range slugs {
		tenantCfg := cfg
		tenantCfg.Database = fmt.Sprintf("%s_%s", cfg.Database, slug)

		log.Info("Migrating tenant database", zap.String("tenant", slug), zap.String("database", tenantCfg.Database))

		if err := migrateDatabase(tenantCfg.BuildURI(), cfg.Migrations.Path, log); err != nil {
			return fmt.Errorf("migration failed for tenant %q: %w", slug, err)
		}
	}

	return nil
}

func migrateDatabase(dbURL, migrationsPath string, log *zap.Logger) error {
	sourcePath := "file://" + migrationsPath

	m, err := migrate.New(sourcePath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Warn("Failed to close migrator source", zap.Error(sourceErr))
		}
		if dbErr != nil {
			log.Warn("Failed to close migrator database", zap.Error(dbErr))
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("Database is up to date, no migrations needed")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, versionErr := m.Version()
	if versionErr != nil {
		log.Warn("Failed to get migration version", zap.Error(versionErr))
	}
	log.Info("Migrations applied successfully",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}
