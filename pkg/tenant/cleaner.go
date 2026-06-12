package tenant

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.uber.org/zap"
)

// Cleaner performs cleanup when a tenant is deleted.
// Register implementations in the "tenant_cleaners" fx group.
type Cleaner interface {
	CleanupTenant(ctx context.Context, slug string) error
}

// mongoCleaner drops tenant databases during tenant cleanup.
type mongoCleaner struct {
	admin mongo.Admin
	log   *zap.Logger
}

// newMongoCleaner creates a Cleaner that drops tenant databases.
func newMongoCleaner(admin mongo.Admin, log *zap.Logger) *mongoCleaner {
	return &mongoCleaner{admin: admin, log: log}
}

// CleanupTenant drops the database for the given tenant slug.
func (c *mongoCleaner) CleanupTenant(ctx context.Context, slug string) error {
	db := c.admin.GetDatabase()
	dbName := fmt.Sprintf("%s_%s", db.Name(), slug)

	c.log.Info("Dropping tenant database", zap.String("tenant", slug), zap.String("database", dbName))

	if err := db.Client().Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop tenant database %q: %w", dbName, err)
	}

	c.log.Info("Tenant database dropped", zap.String("tenant", slug), zap.String("database", dbName))
	return nil
}
