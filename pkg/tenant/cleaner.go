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

// mongoCleaner drops tenant databases during tenant cleanup.
type mongoCleaner struct {
	database *mongodriver.Database
	log      *zap.Logger
}

// NewMongoCleaner creates a Cleaner that drops tenant databases.
func NewMongoCleaner(database *mongodriver.Database, log *zap.Logger) Cleaner {
	return &mongoCleaner{database: database, log: log}
}

// CleanupTenant drops the database for the given tenant slug.
func (c *mongoCleaner) CleanupTenant(ctx context.Context, slug string) error {
	dbName := fmt.Sprintf("%s_%s", c.database.Name(), slug)

	c.log.Info("Dropping tenant database", zap.String("tenant", slug), zap.String("database", dbName))

	if err := c.database.Client().Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop tenant database %q: %w", dbName, err)
	}

	c.log.Info("Tenant database dropped", zap.String("tenant", slug), zap.String("database", dbName))
	return nil
}
