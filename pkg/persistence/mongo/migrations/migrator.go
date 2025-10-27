package migrations

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Migrator interface {
	// Up runs all available migrations from the specified path
	Up(collectionName string, migrationsPath string) error
	// UpFromFS runs all available migrations from embedded filesystem
	UpFromFS(collectionName string, fsys fs.FS, dirPath string) error
	// Down rolls back all migrations from the specified path
	Down(collectionName string, migrationsPath string) error
	// Steps runs n migrations (positive for up, negative for down) from the specified path
	Steps(collectionName string, migrationsPath string, n int) error
	// Version returns the current migration version for the specified path
	Version(collectionName string, migrationsPath string) (uint, bool, error)
}

type migrator struct {
	database       *mongodriver.Database
	log            *zap.Logger
	lockingTimeout int
}

func newMigrator(database *mongodriver.Database, log *zap.Logger, lockingTimeout int) (Migrator, error) {
	return &migrator{
		database:       database,
		log:            log,
		lockingTimeout: lockingTimeout,
	}, nil
}

func (m *migrator) createMigrateInstance(collectionName string, migrationsPath string) (*migrate.Migrate, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if migrationsPath == "" {
		return nil, fmt.Errorf("migrations path is required")
	}

	driver, err := mongodb.WithInstance(m.database.Client(), &mongodb.Config{
		DatabaseName:         m.database.Name(),
		MigrationsCollection: collectionName,
		Locking: mongodb.Locking{
			Enabled: true,
			Timeout: m.lockingTimeout,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb driver: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	mi, err := migrate.NewWithDatabaseInstance(sourceURL, m.database.Name(), driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return mi, nil
}

func (m *migrator) createMigrateInstanceFromFS(collectionName string, fsys fs.FS, dirPath string) (*migrate.Migrate, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if fsys == nil {
		return nil, fmt.Errorf("filesystem is required")
	}

	driver, err := mongodb.WithInstance(m.database.Client(), &mongodb.Config{
		DatabaseName:         m.database.Name(),
		MigrationsCollection: collectionName,
		Locking: mongodb.Locking{
			Enabled: true,
			Timeout: m.lockingTimeout, // timeout in minutes
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb driver: %w", err)
	}

	sourceDriver, err := iofs.New(fsys, dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs source: %w", err)
	}

	mi, err := migrate.NewWithInstance("iofs", sourceDriver, m.database.Name(), driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance with source: %w", err)
	}

	return mi, nil
}

func (m *migrator) Up(collectionName string, migrationsPath string) error {
	m.log.Info("running migrations up",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath))

	mi, err := m.createMigrateInstance(collectionName, migrationsPath)
	if err != nil {
		return err
	}

	err = mi.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		m.log.Info("no migrations to apply")
		return nil
	}

	version, dirty, err := mi.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.log.Info("migrations completed successfully",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath),
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}

func (m *migrator) UpFromFS(collectionName string, fsys fs.FS, dirPath string) error {
	m.log.Info("running migrations up from embedded FS",
		zap.String("collection", collectionName),
		zap.String("dir", dirPath))

	mi, err := m.createMigrateInstanceFromFS(collectionName, fsys, dirPath)
	if err != nil {
		return err
	}

	err = mi.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		m.log.Info("no migrations to apply")
		return nil
	}

	version, dirty, err := mi.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.log.Info("migrations completed successfully",
		zap.String("collection", collectionName),
		zap.String("dir", dirPath),
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}

func (m *migrator) Down(collectionName string, migrationsPath string) error {
	m.log.Warn("rolling back all migrations",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath))

	mi, err := m.createMigrateInstance(collectionName, migrationsPath)
	if err != nil {
		return err
	}

	err = mi.Down()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations down: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		m.log.Info("no migrations to rollback")
		return nil
	}

	m.log.Info("migrations rolled back successfully",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath))
	return nil
}

func (m *migrator) Steps(collectionName string, migrationsPath string, n int) error {
	m.log.Info("running migration steps",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath),
		zap.Int("steps", n))

	mi, err := m.createMigrateInstance(collectionName, migrationsPath)
	if err != nil {
		return err
	}

	err = mi.Steps(n)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migration steps: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		m.log.Info("no migrations to apply")
		return nil
	}

	version, dirty, err := mi.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.log.Info("migration steps completed",
		zap.String("collection", collectionName),
		zap.String("path", migrationsPath),
		zap.Int("steps", n),
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}

func (m *migrator) Version(collectionName string, migrationsPath string) (uint, bool, error) {
	mi, err := m.createMigrateInstance(collectionName, migrationsPath)
	if err != nil {
		return 0, false, err
	}

	version, dirty, err := mi.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}
