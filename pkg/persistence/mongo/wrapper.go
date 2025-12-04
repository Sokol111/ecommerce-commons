package mongo

import (
	"context"
	"fmt"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Middleware represents a decorator that wraps context handling for database operations
type Middleware func(ctx context.Context, next func(context.Context) error) error

// WrapperOption is a functional option for configuring CollectionWrapper
type WrapperOption func(*wrapperOptions)

type wrapperOptions struct {
	middlewares []Middleware
}

// WithTimeout adds timeout middleware to the wrapper
func WithTimeout(timeout time.Duration) WrapperOption {
	return func(o *wrapperOptions) {
		o.middlewares = append(o.middlewares, timeoutMiddleware(timeout))
	}
}

// WithMiddleware adds a custom middleware to the wrapper
func WithMiddleware(mw Middleware) WrapperOption {
	return func(o *wrapperOptions) {
		o.middlewares = append(o.middlewares, mw)
	}
}

// chain combines multiple middlewares into a single middleware.
// Middlewares are executed in order: first middleware wraps second, second wraps third, etc.
func chain(middlewares ...Middleware) Middleware {
	return func(ctx context.Context, next func(context.Context) error) error {
		if len(middlewares) == 0 {
			return next(ctx)
		}

		// Build the chain from right to left
		handler := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = wrapMiddleware(middlewares[i], handler)
		}
		return handler(ctx)
	}
}

// wrapMiddleware creates a handler that wraps the next handler with middleware.
// This avoids closure capture issues in the loop.
func wrapMiddleware(mw Middleware, next func(context.Context) error) func(context.Context) error {
	return func(c context.Context) error {
		return mw(c, next)
	}
}

// timeoutMiddleware adds timeout to context for each operation
func timeoutMiddleware(timeout time.Duration) Middleware {
	return func(ctx context.Context, next func(context.Context) error) error {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return next(timeoutCtx)
	}
}

// Compile-time check that CollectionWrapper implements Collection interface
var _ Collection = (*CollectionWrapper)(nil)

type CollectionWrapper struct {
	coll       Collection
	middleware Middleware
}

// NewCollectionWrapper creates a new CollectionWrapper with the specified options.
// Options are applied in order, so middleware execution order matches the option order.
// Returns error if collection is nil.
//
// Example:
//
//	wrapper, err := NewCollectionWrapper(coll,
//	    WithBulkhead(bulkhead),
//	    WithTimeout(5*time.Second),
//	)
func NewCollectionWrapper(coll Collection, opts ...WrapperOption) (*CollectionWrapper, error) {
	if coll == nil {
		return nil, fmt.Errorf("collection is required")
	}

	o := &wrapperOptions{}
	for _, opt := range opts {
		opt(o)
	}

	return &CollectionWrapper{
		coll:       coll,
		middleware: chain(o.middlewares...),
	}, nil
}

// FindOne wraps collection.FindOne with middleware chain
func (w *CollectionWrapper) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	_ = w.middleware(ctx, func(c context.Context) error {
		result = w.coll.FindOne(c, filter, opts...)
		return nil
	})

	if result == nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// Find wraps collection.Find with middleware chain
func (w *CollectionWrapper) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		cursor, opErr = w.coll.Find(c, filter, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return cursor, nil
}

// InsertOne wraps collection.InsertOne with middleware chain
func (w *CollectionWrapper) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongodriver.InsertOneResult, error) {
	var result *mongodriver.InsertOneResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.InsertOne(c, document, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// InsertMany wraps collection.InsertMany with middleware chain
func (w *CollectionWrapper) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongodriver.InsertManyResult, error) {
	var result *mongodriver.InsertManyResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.InsertMany(c, documents, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateOne wraps collection.UpdateOne with middleware chain
func (w *CollectionWrapper) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.UpdateOne(c, filter, update, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateMany wraps collection.UpdateMany with middleware chain
func (w *CollectionWrapper) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.UpdateMany(c, filter, update, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteOne wraps collection.DeleteOne with middleware chain
func (w *CollectionWrapper) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.DeleteOne(c, filter, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteMany wraps collection.DeleteMany with middleware chain
func (w *CollectionWrapper) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongodriver.DeleteResult, error) {
	var result *mongodriver.DeleteResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.DeleteMany(c, filter, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// FindOneAndUpdate wraps collection.FindOneAndUpdate with middleware chain
func (w *CollectionWrapper) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	_ = w.middleware(ctx, func(c context.Context) error {
		result = w.coll.FindOneAndUpdate(c, filter, update, opts...)
		return nil
	})

	if result == nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// FindOneAndReplace wraps collection.FindOneAndReplace with middleware chain
func (w *CollectionWrapper) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	_ = w.middleware(ctx, func(c context.Context) error {
		result = w.coll.FindOneAndReplace(c, filter, replacement, opts...)
		return nil
	})

	if result == nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// FindOneAndDelete wraps collection.FindOneAndDelete with middleware chain
func (w *CollectionWrapper) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongodriver.SingleResult {
	var result *mongodriver.SingleResult

	_ = w.middleware(ctx, func(c context.Context) error {
		result = w.coll.FindOneAndDelete(c, filter, opts...)
		return nil
	})

	if result == nil {
		return &mongodriver.SingleResult{}
	}
	return result
}

// Aggregate wraps collection.Aggregate with middleware chain
func (w *CollectionWrapper) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongodriver.Cursor, error) {
	var cursor *mongodriver.Cursor
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		cursor, opErr = w.coll.Aggregate(c, pipeline, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return cursor, nil
}

// CountDocuments wraps collection.CountDocuments with middleware chain
func (w *CollectionWrapper) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	var count int64
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		count, opErr = w.coll.CountDocuments(c, filter, opts...)
		return opErr
	})

	if err != nil {
		return 0, err
	}
	return count, nil
}

// Distinct wraps collection.Distinct with middleware chain
func (w *CollectionWrapper) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	var values []interface{}
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		values, opErr = w.coll.Distinct(c, fieldName, filter, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return values, nil
}

// ReplaceOne wraps collection.ReplaceOne with middleware chain
func (w *CollectionWrapper) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongodriver.UpdateResult, error) {
	var result *mongodriver.UpdateResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.ReplaceOne(c, filter, replacement, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// Indexes returns the index view for the collection (no middleware needed for view access)
func (w *CollectionWrapper) Indexes() mongodriver.IndexView {
	return w.coll.Indexes()
}

// Drop wraps collection.Drop with middleware chain
func (w *CollectionWrapper) Drop(ctx context.Context) error {
	return w.middleware(ctx, func(c context.Context) error {
		return w.coll.Drop(c)
	})
}

// Name returns the collection name (no middleware needed)
func (w *CollectionWrapper) Name() string {
	return w.coll.Name()
}

// Database returns the database (no middleware needed)
func (w *CollectionWrapper) Database() *mongodriver.Database {
	return w.coll.Database()
}

// BulkWrite wraps collection.BulkWrite with middleware chain
func (w *CollectionWrapper) BulkWrite(ctx context.Context, models []mongodriver.WriteModel, opts ...*options.BulkWriteOptions) (*mongodriver.BulkWriteResult, error) {
	var result *mongodriver.BulkWriteResult
	var opErr error

	err := w.middleware(ctx, func(c context.Context) error {
		result, opErr = w.coll.BulkWrite(c, models, opts...)
		return opErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
