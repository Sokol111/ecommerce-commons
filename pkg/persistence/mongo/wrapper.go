package mongo

import (
	"context"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionWrapper struct {
	coll     Collection
	timeout  time.Duration
	bulkhead *Bulkhead
}

func NewCollectionWrapper(coll Collection, timeout time.Duration, bulkhead *Bulkhead) *CollectionWrapper {
	return &CollectionWrapper{
		coll:     coll,
		timeout:  timeout,
		bulkhead: bulkhead,
	}
}

// withTimeout creates context with query timeout
func (w *CollectionWrapper) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, w.timeout)
}

// withBulkhead executes a function with bulkhead protection
func (w *CollectionWrapper) withBulkhead(ctx context.Context, fn func() error) error {
	return w.bulkhead.Execute(ctx, fn)
}

// FindOne wraps collection.FindOne with automatic timeout and bulkhead
func (w *CollectionWrapper) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result = w.coll.FindOne(timeoutCtx, filter, opts...)
		return nil
	})

	if err != nil {
		// Return error as SingleResult
		return &mongodriver.SingleResult{}
	}
	return result
}

// Find wraps collection.Find with automatic timeout and bulkhead
func (w *CollectionWrapper) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var findErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		cursor, findErr = w.coll.Find(timeoutCtx, filter, opts...)
		return findErr
	})

	if err != nil {
		return nil, err
	}
	return cursor, findErr
}

// InsertOne wraps collection.InsertOne with automatic timeout and bulkhead
func (w *CollectionWrapper) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongodriver.InsertOneResult, error) {
	var result *mongodriver.InsertOneResult
	var insertErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, insertErr = w.coll.InsertOne(timeoutCtx, document, opts...)
		return insertErr
	})

	if err != nil {
		return nil, err
	}
	return result, insertErr
}

// InsertMany wraps collection.InsertMany with automatic timeout and bulkhead
func (w *CollectionWrapper) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongodriver.InsertManyResult, error) {
	var result *mongodriver.InsertManyResult
	var insertErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, insertErr = w.coll.InsertMany(timeoutCtx, documents, opts...)
		return insertErr
	})

	if err != nil {
		return nil, err
	}
	return result, insertErr
}

// UpdateOne wraps collection.UpdateOne with automatic timeout and bulkhead
func (w *CollectionWrapper) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var updateErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, updateErr = w.coll.UpdateOne(timeoutCtx, filter, update, opts...)
		return updateErr
	})

	if err != nil {
		return nil, err
	}
	return result, updateErr
}

// UpdateMany wraps collection.UpdateMany with automatic timeout and bulkhead
func (w *CollectionWrapper) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var updateErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, updateErr = w.coll.UpdateMany(timeoutCtx, filter, update, opts...)
		return updateErr
	})

	if err != nil {
		return nil, err
	}
	return result, updateErr
}

// DeleteOne wraps collection.DeleteOne with automatic timeout and bulkhead
func (w *CollectionWrapper) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var deleteErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, deleteErr = w.coll.DeleteOne(timeoutCtx, filter, opts...)
		return deleteErr
	})

	if err != nil {
		return nil, err
	}
	return result, deleteErr
}

// DeleteMany wraps collection.DeleteMany with automatic timeout and bulkhead
func (w *CollectionWrapper) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var deleteErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, deleteErr = w.coll.DeleteMany(timeoutCtx, filter, opts...)
		return deleteErr
	})

	if err != nil {
		return nil, err
	}
	return result, deleteErr
}

// FindOneAndUpdate wraps collection.FindOneAndUpdate with automatic timeout and bulkhead
func (w *CollectionWrapper) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result = w.coll.FindOneAndUpdate(timeoutCtx, filter, update, opts...)
		return nil
	})

	if err != nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// FindOneAndReplace wraps collection.FindOneAndReplace with automatic timeout and bulkhead
func (w *CollectionWrapper) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result = w.coll.FindOneAndReplace(timeoutCtx, filter, replacement, opts...)
		return nil
	})

	if err != nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// FindOneAndDelete wraps collection.FindOneAndDelete with automatic timeout and bulkhead
func (w *CollectionWrapper) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result = w.coll.FindOneAndDelete(timeoutCtx, filter, opts...)
		return nil
	})

	if err != nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// Aggregate wraps collection.Aggregate with automatic timeout and bulkhead
func (w *CollectionWrapper) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var aggErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		cursor, aggErr = w.coll.Aggregate(timeoutCtx, pipeline, opts...)
		return aggErr
	})

	if err != nil {
		return nil, err
	}
	return cursor, aggErr
}

// CountDocuments wraps collection.CountDocuments with automatic timeout and bulkhead
func (w *CollectionWrapper) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	var count int64
	var countErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		count, countErr = w.coll.CountDocuments(timeoutCtx, filter, opts...)
		return countErr
	})

	if err != nil {
		return 0, err
	}
	return count, countErr
}

// Distinct wraps collection.Distinct with automatic timeout and bulkhead
func (w *CollectionWrapper) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	var values []interface{}
	var distinctErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		values, distinctErr = w.coll.Distinct(timeoutCtx, fieldName, filter, opts...)
		return distinctErr
	})

	if err != nil {
		return nil, err
	}
	return values, distinctErr
}

// ReplaceOne wraps collection.ReplaceOne with automatic timeout and bulkhead
func (w *CollectionWrapper) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var replaceErr error

	err := w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		result, replaceErr = w.coll.ReplaceOne(timeoutCtx, filter, replacement, opts...)
		return replaceErr
	})

	if err != nil {
		return nil, err
	}
	return result, replaceErr
}

// Indexes returns the index view for the collection (no timeout needed for view access)
func (w *CollectionWrapper) Indexes() mongodriver.IndexView {
	return w.coll.Indexes()
}

// Drop wraps collection.Drop with automatic timeout and bulkhead
func (w *CollectionWrapper) Drop(ctx context.Context) error {
	return w.withBulkhead(ctx, func() error {
		timeoutCtx, cancel := w.withTimeout(ctx)
		defer cancel()
		return w.coll.Drop(timeoutCtx)
	})
}

// Name returns the collection name (no timeout needed)
func (w *CollectionWrapper) Name() string {
	return w.coll.Name()
}

// Database returns the database (no timeout needed)
func (w *CollectionWrapper) Database() *mongodriver.Database {
	return w.coll.Database()
}
