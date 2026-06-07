package mongo

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// DatabaseResolver resolves a database name suffix from the request context.
// Used by multi-tenant collection providers to determine the target database.
type DatabaseResolver func(ctx context.Context) string

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

// dynamicCollectionProvider resolves a collection in the context-specific database
// using the database-per-context strategy. Each context gets its own database
// named "{baseDatabaseName}_{suffix}" where suffix is resolved by DatabaseResolver.
type dynamicCollectionProvider struct {
	client           *mongodriver.Client
	baseDatabaseName string
	collectionName   string
	resolver         DatabaseResolver
}

func newDynamicCollectionProvider(admin Admin, collectionName string, resolver DatabaseResolver) collectionProvider {
	db := admin.GetDatabase()
	return &dynamicCollectionProvider{
		client:           db.Client(),
		baseDatabaseName: db.Name(),
		collectionName:   collectionName,
		resolver:         resolver,
	}
}

// GetCollection resolves the collection for the current context.
func (d *dynamicCollectionProvider) GetCollection(ctx context.Context) *mongodriver.Collection {
	suffix := d.resolver(ctx)
	return d.client.Database(fmt.Sprintf("%s_%s", d.baseDatabaseName, suffix)).Collection(d.collectionName)
}
