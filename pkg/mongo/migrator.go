package mongo

import (
	"context"

	"go.uber.org/zap"
)

// MigrationRunner represents a component that can run database migrations.
type MigrationRunner interface {
	Run(ctx context.Context) error
}

// SingleMigrationRunner migrates a single database.
type SingleMigrationRunner struct {
	uri            string
	migrationsPath string
	log            *zap.Logger
}

// NewSingleMigrationRunner creates SingleMigrationRunner.
func NewSingleMigrationRunner(cfg Config, log *zap.Logger) *SingleMigrationRunner {
	return &SingleMigrationRunner{
		uri:            cfg.BuildURI(),
		migrationsPath: cfg.Migrations.Path,
		log:            log,
	}
}

// Run runs the migrations for the database.
func (r *SingleMigrationRunner) Run(_ context.Context) error {
	return MigrateDatabase(r.uri, r.migrationsPath, r.log)
}
