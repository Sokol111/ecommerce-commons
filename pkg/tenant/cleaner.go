package tenant

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

// Cleaner performs cleanup when a tenant is deleted.
// Register implementations in the "tenant_cleaners" fx group.
type Cleaner interface {
	CleanupTenant(ctx context.Context, slug string) error
}

// mongoCleanupCleaner drops tenant databases during tenant cleanup.
type mongoCleanupCleaner struct {
	db  *mongodriver.Database
	log *zap.Logger
}

// newMongoCleanupCleaner creates a Cleaner that drops tenant databases.
func newMongoCleanupCleaner(db *mongodriver.Database, log *zap.Logger) *mongoCleanupCleaner {
	return &mongoCleanupCleaner{db: db, log: log}
}

// CleanupTenant drops the database for the given tenant slug.
func (c *mongoCleanupCleaner) CleanupTenant(ctx context.Context, slug string) error {
	dbName := fmt.Sprintf("%s_%s", c.db.Name(), slug)

	c.log.Info("Dropping tenant database", zap.String("tenant", slug), zap.String("database", dbName))

	if err := c.db.Client().Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop tenant database %q: %w", dbName, err)
	}

	c.log.Info("Tenant database dropped", zap.String("tenant", slug), zap.String("database", dbName))
	return nil
}
