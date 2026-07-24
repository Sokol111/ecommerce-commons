package tenant

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// MultiTenantCollectionProvider resolves a collection in the context-specific database
// using the database-per-context strategy. Each context gets its own database
// named "{baseDatabaseName}_{suffix}" where suffix is resolved by DatabaseResolver.
type MultiTenantCollectionProvider struct {
	client           *mongodriver.Client
	baseDatabaseName string
	collectionName   string
}

func NewMultiTenantCollectionProvider(database *mongodriver.Database, collectionName string) *MultiTenantCollectionProvider {
	return &MultiTenantCollectionProvider{
		client:           database.Client(),
		baseDatabaseName: database.Name(),
		collectionName:   collectionName,
	}
}

// GetCollection resolves the collection for the current context.
func (d *MultiTenantCollectionProvider) GetCollection(ctx context.Context) *mongodriver.Collection {
	tenant := MustSlugFromContext(ctx)
	return d.client.Database(fmt.Sprintf("%s_%s", d.baseDatabaseName, tenant)).Collection(d.collectionName)
}
