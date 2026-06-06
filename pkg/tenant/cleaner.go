package tenant

import "context"

// Cleaner performs cleanup when a tenant is deleted.
// Register implementations in the "tenant_cleaners" fx group.
type Cleaner interface {
	CleanupTenant(ctx context.Context, slug string) error
}
