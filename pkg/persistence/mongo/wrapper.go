package mongo

import (
	"context"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Middleware represents a decorator that wraps context transformation for database operations.
// Middleware should only modify context (e.g., add timeout, tracing), not return errors.
type Middleware func(ctx context.Context, next func(context.Context))

// WrapperOption is a functional option for configuring CollectionWrapper.
type WrapperOption func(*wrapperOptions)

type wrapperOptions struct {
	middlewares []Middleware
}

// WithTimeout adds timeout middleware to the wrapper.
func WithTimeout(timeout time.Duration) WrapperOption {
	return func(o *wrapperOptions) {
		o.middlewares = append(o.middlewares, timeoutMiddleware(timeout))
	}
}

// WithMiddleware adds a custom middleware to the wrapper.
func WithMiddleware(mw Middleware) WrapperOption {
	return func(o *wrapperOptions) {
		o.middlewares = append(o.middlewares, mw)
	}
}

// chain combines multiple middlewares into a single middleware.
// Middlewares are executed in order: first middleware wraps second, second wraps third, etc.
// The chain is built once at creation time, not on every call.
func chain(middlewares ...Middleware) Middleware {
	if len(middlewares) == 0 {
		return func(ctx context.Context, next func(context.Context)) {
			next(ctx)
		}
	}

	if len(middlewares) == 1 {
		return middlewares[0]
	}

	// Build the chain once, from right to left
	// Result: mw[0] wraps mw[1] wraps ... wraps mw[n-1]
	chained := middlewares[len(middlewares)-1]
	for i := len(middlewares) - 2; i >= 0; i-- {
		chained = composeMiddleware(middlewares[i], chained)
	}
	return chained
}

// composeMiddleware combines two middlewares into one.
// The outer middleware wraps the inner middleware.
func composeMiddleware(outer, inner Middleware) Middleware {
	return func(ctx context.Context, next func(context.Context)) {
		outer(ctx, func(c context.Context) {
			inner(c, next)
		})
	}
}

// timeoutMiddleware adds timeout to context for each operation
func timeoutMiddleware(timeout time.Duration) Middleware {
	return func(ctx context.Context, next func(context.Context)) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		next(timeoutCtx)
	}
}

// UnwrapCollection unwraps a Collection to the underlying *mongo.Collection.
// This is useful when you need direct access to the driver's collection
// for operations not exposed through the Collection interface.
// Panics if the underlying collection is not *mongo.Collection (should never happen
// when using collections from Mongo.GetCollection/GetCollectionWithOptions).
func UnwrapCollection(coll Collection) *mongodriver.Collection {
	if wrapper, ok := coll.(*collectionWrapper); ok {
		return wrapper.coll.(*mongodriver.Collection) //nolint:errcheck // type is guaranteed by constructor
	}
	return coll.(*mongodriver.Collection) //nolint:errcheck // direct mongo.Collection
}

// Compile-time check that collectionWrapper implements Collection interface.
var _ Collection = (*collectionWrapper)(nil)

type collectionWrapper struct {
	coll       Collection
	middleware Middleware
}

// newCollectionWrapper creates a new collectionWrapper with the specified options.
// Options are applied in order, so middleware execution order matches the option order.
// Panics if collection is nil (programmer error).
func newCollectionWrapper(coll Collection, opts ...WrapperOption) *collectionWrapper {
	if coll == nil {
		panic("mongo: collection is nil")
	}

	o := &wrapperOptions{}
	for _, opt := range opts {
		opt(o)
	}

	return &collectionWrapper{
		coll:       coll,
		middleware: chain(o.middlewares...),
	}
}

// FindOne wraps collection.FindOne with middleware chain
func (w *collectionWrapper) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult
	w.middleware(ctx, func(c context.Context) {
		result = w.coll.FindOne(c, filter, opts...)
	})
	return result
}

// Find wraps collection.Find with middleware chain
func (w *collectionWrapper) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var err error
	w.middleware(ctx, func(c context.Context) {
		cursor, err = w.coll.Find(c, filter, opts...)
	})
	return cursor, err
}

// InsertOne wraps collection.InsertOne with middleware chain
func (w *collectionWrapper) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongodriver.InsertOneResult, error) {
	var result *mongodriver.InsertOneResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.InsertOne(c, document, opts...)
	})
	return result, err
}

// InsertMany wraps collection.InsertMany with middleware chain
func (w *collectionWrapper) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongodriver.InsertManyResult, error) {
	var result *mongodriver.InsertManyResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.InsertMany(c, documents, opts...)
	})
	return result, err
}

// UpdateOne wraps collection.UpdateOne with middleware chain
func (w *collectionWrapper) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.UpdateOne(c, filter, update, opts...)
	})
	return result, err
}

// UpdateMany wraps collection.UpdateMany with middleware chain
func (w *collectionWrapper) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.UpdateMany(c, filter, update, opts...)
	})
	return result, err
}

// DeleteOne wraps collection.DeleteOne with middleware chain
func (w *collectionWrapper) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.DeleteOne(c, filter, opts...)
	})
	return result, err
}

// DeleteMany wraps collection.DeleteMany with middleware chain
func (w *collectionWrapper) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.DeleteMany(c, filter, opts...)
	})
	return result, err
}

// FindOneAndUpdate wraps collection.FindOneAndUpdate with middleware chain
func (w *collectionWrapper) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult
	w.middleware(ctx, func(c context.Context) {
		result = w.coll.FindOneAndUpdate(c, filter, update, opts...)
	})
	return result
}

// FindOneAndReplace wraps collection.FindOneAndReplace with middleware chain
func (w *collectionWrapper) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult
	w.middleware(ctx, func(c context.Context) {
		result = w.coll.FindOneAndReplace(c, filter, replacement, opts...)
	})
	return result
}

// FindOneAndDelete wraps collection.FindOneAndDelete with middleware chain
func (w *collectionWrapper) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult
	w.middleware(ctx, func(c context.Context) {
		result = w.coll.FindOneAndDelete(c, filter, opts...)
	})
	return result
}

// Aggregate wraps collection.Aggregate with middleware chain
func (w *collectionWrapper) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var err error
	w.middleware(ctx, func(c context.Context) {
		cursor, err = w.coll.Aggregate(c, pipeline, opts...)
	})
	return cursor, err
}

// CountDocuments wraps collection.CountDocuments with middleware chain
func (w *collectionWrapper) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	var count int64
	var err error
	w.middleware(ctx, func(c context.Context) {
		count, err = w.coll.CountDocuments(c, filter, opts...)
	})
	return count, err
}

// Distinct wraps collection.Distinct with middleware chain
func (w *collectionWrapper) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	var values []interface{}
	var err error
	w.middleware(ctx, func(c context.Context) {
		values, err = w.coll.Distinct(c, fieldName, filter, opts...)
	})
	return values, err
}

// ReplaceOne wraps collection.ReplaceOne with middleware chain
func (w *collectionWrapper) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.ReplaceOne(c, filter, replacement, opts...)
	})
	return result, err
}

// Indexes returns the index view for the collection (no middleware needed for view access).
func (w *collectionWrapper) Indexes() mongodriver.IndexView {
	return w.coll.Indexes()
}

// Drop wraps collection.Drop with middleware chain
func (w *collectionWrapper) Drop(ctx context.Context) error {
	var err error
	w.middleware(ctx, func(c context.Context) {
		err = w.coll.Drop(c)
	})
	return err
}

// Name returns the collection name (no middleware needed).
func (w *collectionWrapper) Name() string {
	return w.coll.Name()
}

// Database returns the database (no middleware needed).
func (w *collectionWrapper) Database() *mongodriver.Database {
	return w.coll.Database()
}

// BulkWrite wraps collection.BulkWrite with middleware chain.
func (w *collectionWrapper) BulkWrite(ctx context.Context, models []mongodriver.WriteModel, opts ...*options.BulkWriteOptions) (*mongodriver.BulkWriteResult, error) {
	var result *mongodriver.BulkWriteResult
	var err error
	w.middleware(ctx, func(c context.Context) {
		result, err = w.coll.BulkWrite(c, models, opts...)
	})
	return result, err
}
