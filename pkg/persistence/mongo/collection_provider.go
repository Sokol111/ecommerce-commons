package mongo

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// collectionProvider resolves a MongoDB collection from context.
// Used by GenericRepository to support both fixed and tenant-aware collections.
type collectionProvider interface {
	GetCollection(ctx context.Context) *mongodriver.Collection
}

// staticCollectionProvider always returns the same collection.
type staticCollectionProvider struct {
	coll *mongodriver.Collection
}

func newStaticCollectionProvider(coll *mongodriver.Collection) collectionProvider {
	return &staticCollectionProvider{coll: coll}
}

func (s *staticCollectionProvider) GetCollection(_ context.Context) *mongodriver.Collection {
	return s.coll
}

// tenantCollectionProvider resolves a collection in the tenant-specific database
// using the database-per-tenant strategy. Each tenant gets its own database
// named "{baseDatabaseName}_{tenantSlug}".
type tenantCollectionProvider struct {
	client           *mongodriver.Client
	baseDatabaseName string
	collectionName   string
}

func newTenantCollectionProvider(admin Admin, collectionName string) collectionProvider {
	db := admin.GetDatabase()
	return &tenantCollectionProvider{
		client:           db.Client(),
		baseDatabaseName: db.Name(),
		collectionName:   collectionName,
	}
}

// GetCollection resolves the collection for the tenant in the current context.
func (t *tenantCollectionProvider) GetCollection(ctx context.Context) *mongodriver.Collection {
	slug := tenant.MustSlugFromContext(ctx)
	return t.client.Database(fmt.Sprintf("%s_%s", t.baseDatabaseName, slug)).Collection(t.collectionName)
}
