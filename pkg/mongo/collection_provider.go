package mongo

import (
	"context"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

// CollectionProvider resolves a MongoDB collection from context.
// Used by GenericRepository to support both fixed and tenant-aware collections.
type CollectionProvider interface {
	GetCollection(ctx context.Context) *mongodriver.Collection
}

// StaticCollectionProvider always returns the same collection.
type StaticCollectionProvider struct {
	coll *mongodriver.Collection
}

func NewStaticCollectionProvider(coll *mongodriver.Collection) *StaticCollectionProvider {
	return &StaticCollectionProvider{coll: coll}
}

func (s *StaticCollectionProvider) GetCollection(_ context.Context) *mongodriver.Collection {
	return s.coll
}
