package mongo

import (
	"context"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionWrapper struct {
	coll    Collection
	timeout time.Duration
}

func NewCollectionWrapper(coll Collection, timeout time.Duration) *CollectionWrapper {
	return &CollectionWrapper{
		coll:    coll,
		timeout: timeout,
	}
}

// withTimeout creates context with query timeout
func (w *CollectionWrapper) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, w.timeout)
}

// FindOne wraps collection.FindOne with automatic timeout
func (w *CollectionWrapper) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongodriver.SingleResult {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.FindOne(timeoutCtx, filter, opts...)
}

// Find wraps collection.Find with automatic timeout
func (w *CollectionWrapper) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongodriver.Cursor, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.Find(timeoutCtx, filter, opts...)
}

// InsertOne wraps collection.InsertOne with automatic timeout
func (w *CollectionWrapper) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongodriver.InsertOneResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.InsertOne(timeoutCtx, document, opts...)
}

// InsertMany wraps collection.InsertMany with automatic timeout
func (w *CollectionWrapper) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongodriver.InsertManyResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.InsertMany(timeoutCtx, documents, opts...)
}

// UpdateOne wraps collection.UpdateOne with automatic timeout
func (w *CollectionWrapper) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.UpdateOne(timeoutCtx, filter, update, opts...)
}

// UpdateMany wraps collection.UpdateMany with automatic timeout
func (w *CollectionWrapper) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.UpdateMany(timeoutCtx, filter, update, opts...)
}

// DeleteOne wraps collection.DeleteOne with automatic timeout
func (w *CollectionWrapper) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.DeleteOne(timeoutCtx, filter, opts...)
}

// DeleteMany wraps collection.DeleteMany with automatic timeout
func (w *CollectionWrapper) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.DeleteMany(timeoutCtx, filter, opts...)
}

// FindOneAndUpdate wraps collection.FindOneAndUpdate with automatic timeout
func (w *CollectionWrapper) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongodriver.SingleResult {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.FindOneAndUpdate(timeoutCtx, filter, update, opts...)
}

// FindOneAndReplace wraps collection.FindOneAndReplace with automatic timeout
func (w *CollectionWrapper) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongodriver.SingleResult {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.FindOneAndReplace(timeoutCtx, filter, replacement, opts...)
}

// FindOneAndDelete wraps collection.FindOneAndDelete with automatic timeout
func (w *CollectionWrapper) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongodriver.SingleResult {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.FindOneAndDelete(timeoutCtx, filter, opts...)
}

// Aggregate wraps collection.Aggregate with automatic timeout
func (w *CollectionWrapper) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongodriver.Cursor, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.Aggregate(timeoutCtx, pipeline, opts...)
}

// CountDocuments wraps collection.CountDocuments with automatic timeout
func (w *CollectionWrapper) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.CountDocuments(timeoutCtx, filter, opts...)
}

// Distinct wraps collection.Distinct with automatic timeout
func (w *CollectionWrapper) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.Distinct(timeoutCtx, fieldName, filter, opts...)
}

// ReplaceOne wraps collection.ReplaceOne with automatic timeout
func (w *CollectionWrapper) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongodriver.UpdateResult, error) {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.ReplaceOne(timeoutCtx, filter, replacement, opts...)
}

// Indexes returns the index view for the collection (no timeout needed for view access)
func (w *CollectionWrapper) Indexes() mongodriver.IndexView {
	return w.coll.Indexes()
}

// Drop wraps collection.Drop with automatic timeout
func (w *CollectionWrapper) Drop(ctx context.Context) error {
	timeoutCtx, cancel := w.withTimeout(ctx)
	defer cancel()
	return w.coll.Drop(timeoutCtx)
}

// Name returns the collection name (no timeout needed)
func (w *CollectionWrapper) Name() string {
	return w.coll.Name()
}

// Database returns the database (no timeout needed)
func (w *CollectionWrapper) Database() *mongodriver.Database {
	return w.coll.Database()
}
