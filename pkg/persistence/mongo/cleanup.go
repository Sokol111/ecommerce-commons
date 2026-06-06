package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	"go.uber.org/zap"
)

type tenantCleanupRunner struct {
	admin Admin
	cfg   Config
	log   *zap.Logger
}

// NewTenantCleanupCleaner creates a tenant.Cleaner that drops tenant databases.
func NewTenantCleanupCleaner(admin Admin, cfg Config, log *zap.Logger) tenant.Cleaner {
	return &tenantCleanupRunner{admin: admin, cfg: cfg, log: log}
}

func (r *tenantCleanupRunner) CleanupTenant(ctx context.Context, slug string) error {
	dbName := fmt.Sprintf("%s_%s", r.cfg.Database, slug)

	r.log.Info("Dropping tenant database", zap.String("tenant", slug), zap.String("database", dbName))

	if err := r.admin.GetDatabase().Client().Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop tenant database %q: %w", dbName, err)
	}

	r.log.Info("Tenant database dropped", zap.String("tenant", slug), zap.String("database", dbName))
	return nil
}
