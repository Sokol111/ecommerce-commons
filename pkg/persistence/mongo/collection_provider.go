package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// CollectionProvider resolves a MongoDB collection from context.
// Used by GenericRepository to support both fixed and tenant-aware collections.
type CollectionProvider interface {
	GetCollection(ctx context.Context) *mongodriver.Collection
}

// staticCollectionProvider always returns the same collection.
type staticCollectionProvider struct {
	coll *mongodriver.Collection
}

// StaticCollectionProvider returns a CollectionProvider that always returns the same collection.
// Used for non-tenant-aware (single-tenant) repositories.
func StaticCollectionProvider(coll *mongodriver.Collection) CollectionProvider {
	return &staticCollectionProvider{coll: coll}
}

func (s *staticCollectionProvider) GetCollection(_ context.Context) *mongodriver.Collection {
	return s.coll
}

// TenantCollectionProvider resolves a collection in the tenant-specific database
// using the database-per-tenant strategy. Each tenant gets its own database
// named "{baseDatabaseName}_{tenantSlug}".
type TenantCollectionProvider struct {
	client           *mongodriver.Client
	baseDatabaseName string
	collectionName   string
}

// NewTenantCollectionProvider creates a CollectionProvider that resolves the collection
// in the tenant-specific database based on the tenant slug in the context.
func NewTenantCollectionProvider(admin Admin, collectionName string) *TenantCollectionProvider {
	db := admin.GetDatabase()
	return &TenantCollectionProvider{
		client:           db.Client(),
		baseDatabaseName: db.Name(),
		collectionName:   collectionName,
	}
}

// GetCollection resolves the collection for the tenant in the current context.
func (t *TenantCollectionProvider) GetCollection(ctx context.Context) *mongodriver.Collection {
	slug := tenant.MustSlugFromContext(ctx)
	return t.client.Database(fmt.Sprintf("%s_%s", t.baseDatabaseName, slug)).Collection(t.collectionName)
}
