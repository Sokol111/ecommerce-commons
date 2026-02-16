package mongo

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb" // MongoDB driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // File source
	"go.uber.org/zap"
)

// runMigrations applies database migrations if enabled in config.
func runMigrations(cfg Config, log *zap.Logger) error {
	if !cfg.Migrations.Enabled {
		log.Info("Database migrations disabled")
		return nil
	}

	sourcePath := "file://" + cfg.Migrations.Path
	dbURL := cfg.BuildURI()

	log.Info("Running database migrations...",
		zap.String("path", cfg.Migrations.Path),
	)

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
