package mongo

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb" // MongoDB driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // File source
	"go.uber.org/zap"
)

// runMigrations applies database migrations.
func runMigrations(cfg Config, log *zap.Logger) error {
	return MigrateDatabase(cfg.BuildURI(), cfg.Migrations.Path, log)
}

// MigrateDatabase applies file-based migrations to the given MongoDB database URI.
func MigrateDatabase(dbURL, migrationsPath string, log *zap.Logger) error {
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
