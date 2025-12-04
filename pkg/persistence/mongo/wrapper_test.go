package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo/mongomock"
)

func TestNewCollectionWrapper(t *testing.T) {
	t.Run("creates wrapper with no options", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		assert.NotNil(t, wrapper)
		assert.Equal(t, mockColl, wrapper.coll)
	})

	t.Run("panics when collection is nil", func(t *testing.T) {
		assert.Panics(t, func() {
			newCollectionWrapper(nil)
		})
	})

	t.Run("creates wrapper with timeout option", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl, WithTimeout(5*time.Second))

		assert.NotNil(t, wrapper)
		assert.NotNil(t, wrapper.middleware)
	})

	t.Run("creates wrapper with custom middleware", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		customMw := func(ctx context.Context, next func(context.Context)) {
			next(ctx)
		}
		wrapper := newCollectionWrapper(mockColl, WithMiddleware(customMw))

		assert.NotNil(t, wrapper)
		assert.NotNil(t, wrapper.middleware)
	})
}

func TestChainMiddleware(t *testing.T) {
	t.Run("chain with no middlewares calls next directly", func(t *testing.T) {
		called := false
		chained := chain()
		chained(context.Background(), func(ctx context.Context) {
			called = true
		})
		assert.True(t, called)
	})

	t.Run("chain with single middleware", func(t *testing.T) {
		order := []string{}
		mw := func(ctx context.Context, next func(context.Context)) {
			order = append(order, "mw-before")
			next(ctx)
			order = append(order, "mw-after")
		}

		chained := chain(mw)
		chained(context.Background(), func(ctx context.Context) {
			order = append(order, "handler")
		})

		assert.Equal(t, []string{"mw-before", "handler", "mw-after"}, order)
	})

	t.Run("chain with multiple middlewares executes in order", func(t *testing.T) {
		order := []string{}
		mw1 := func(ctx context.Context, next func(context.Context)) {
			order = append(order, "mw1-before")
			next(ctx)
			order = append(order, "mw1-after")
		}
		mw2 := func(ctx context.Context, next func(context.Context)) {
			order = append(order, "mw2-before")
			next(ctx)
			order = append(order, "mw2-after")
		}

		chained := chain(mw1, mw2)
		chained(context.Background(), func(ctx context.Context) {
			order = append(order, "handler")
		})

		assert.Equal(t, []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}, order)
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	t.Run("adds timeout to context", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		mw := timeoutMiddleware(timeout)

		var capturedCtx context.Context
		mw(context.Background(), func(ctx context.Context) {
			capturedCtx = ctx
		})

		deadline, ok := capturedCtx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(timeout), deadline, 50*time.Millisecond)
	})
}

func TestCollectionWrapperFindOne(t *testing.T) {
	t.Run("calls underlying collection FindOne", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}

		mockColl.EXPECT().FindOne(mock.Anything, filter).Return(&mongodriver.SingleResult{})

		result := wrapper.FindOne(ctx, filter)
		assert.NotNil(t, result)
	})

	t.Run("passes options to underlying collection", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		opts := options.FindOne().SetProjection(map[string]interface{}{"name": 1})

		mockColl.EXPECT().FindOne(mock.Anything, filter, mock.Anything).Return(&mongodriver.SingleResult{})

		result := wrapper.FindOne(ctx, filter, opts)
		assert.NotNil(t, result)
	})
}

func TestCollectionWrapperFind(t *testing.T) {
	t.Run("calls underlying collection Find", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "active"}

		mockColl.EXPECT().Find(mock.Anything, filter).Return(nil, nil)

		cursor, err := wrapper.Find(ctx, filter)
		assert.NoError(t, err)
		assert.Nil(t, cursor)
	})

	t.Run("returns error from underlying collection", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "active"}
		expectedErr := errors.New("find error")

		mockColl.EXPECT().Find(mock.Anything, filter).Return(nil, expectedErr)

		cursor, err := wrapper.Find(ctx, filter)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, cursor)
	})
}

func TestCollectionWrapperInsertOne(t *testing.T) {
	t.Run("calls underlying collection InsertOne", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		doc := map[string]interface{}{"name": "test"}
		expectedResult := &mongodriver.InsertOneResult{InsertedID: "123"}

		mockColl.EXPECT().InsertOne(mock.Anything, doc).Return(expectedResult, nil)

		result, err := wrapper.InsertOne(ctx, doc)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("returns error from underlying collection", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		doc := map[string]interface{}{"name": "test"}
		expectedErr := errors.New("insert error")

		mockColl.EXPECT().InsertOne(mock.Anything, doc).Return(nil, expectedErr)

		result, err := wrapper.InsertOne(ctx, doc)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestCollectionWrapperInsertMany(t *testing.T) {
	t.Run("calls underlying collection InsertMany", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		docs := []interface{}{
			map[string]interface{}{"name": "test1"},
			map[string]interface{}{"name": "test2"},
		}
		expectedResult := &mongodriver.InsertManyResult{InsertedIDs: []interface{}{"1", "2"}}

		mockColl.EXPECT().InsertMany(mock.Anything, docs).Return(expectedResult, nil)

		result, err := wrapper.InsertMany(ctx, docs)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperUpdateOne(t *testing.T) {
	t.Run("calls underlying collection UpdateOne", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		update := map[string]interface{}{"$set": map[string]interface{}{"name": "updated"}}
		expectedResult := &mongodriver.UpdateResult{ModifiedCount: 1}

		mockColl.EXPECT().UpdateOne(mock.Anything, filter, update).Return(expectedResult, nil)

		result, err := wrapper.UpdateOne(ctx, filter, update)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperUpdateMany(t *testing.T) {
	t.Run("calls underlying collection UpdateMany", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "pending"}
		update := map[string]interface{}{"$set": map[string]interface{}{"status": "active"}}
		expectedResult := &mongodriver.UpdateResult{ModifiedCount: 5}

		mockColl.EXPECT().UpdateMany(mock.Anything, filter, update).Return(expectedResult, nil)

		result, err := wrapper.UpdateMany(ctx, filter, update)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperDeleteOne(t *testing.T) {
	t.Run("calls underlying collection DeleteOne", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		expectedResult := &mongodriver.DeleteResult{DeletedCount: 1}

		mockColl.EXPECT().DeleteOne(mock.Anything, filter).Return(expectedResult, nil)

		result, err := wrapper.DeleteOne(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperDeleteMany(t *testing.T) {
	t.Run("calls underlying collection DeleteMany", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "deleted"}
		expectedResult := &mongodriver.DeleteResult{DeletedCount: 10}

		mockColl.EXPECT().DeleteMany(mock.Anything, filter).Return(expectedResult, nil)

		result, err := wrapper.DeleteMany(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperFindOneAndUpdate(t *testing.T) {
	t.Run("calls underlying collection FindOneAndUpdate", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		update := map[string]interface{}{"$set": map[string]interface{}{"name": "updated"}}

		mockColl.EXPECT().FindOneAndUpdate(mock.Anything, filter, update).Return(&mongodriver.SingleResult{})

		result := wrapper.FindOneAndUpdate(ctx, filter, update)
		assert.NotNil(t, result)
	})
}

func TestCollectionWrapperFindOneAndReplace(t *testing.T) {
	t.Run("calls underlying collection FindOneAndReplace", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		replacement := map[string]interface{}{"name": "replaced", "status": "active"}

		mockColl.EXPECT().FindOneAndReplace(mock.Anything, filter, replacement).Return(&mongodriver.SingleResult{})

		result := wrapper.FindOneAndReplace(ctx, filter, replacement)
		assert.NotNil(t, result)
	})
}

func TestCollectionWrapperFindOneAndDelete(t *testing.T) {
	t.Run("calls underlying collection FindOneAndDelete", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}

		mockColl.EXPECT().FindOneAndDelete(mock.Anything, filter).Return(&mongodriver.SingleResult{})

		result := wrapper.FindOneAndDelete(ctx, filter)
		assert.NotNil(t, result)
	})
}

func TestCollectionWrapperAggregate(t *testing.T) {
	t.Run("calls underlying collection Aggregate", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		pipeline := []map[string]interface{}{
			{"$match": map[string]interface{}{"status": "active"}},
		}

		mockColl.EXPECT().Aggregate(mock.Anything, pipeline).Return(nil, nil)

		cursor, err := wrapper.Aggregate(ctx, pipeline)
		assert.NoError(t, err)
		assert.Nil(t, cursor)
	})
}

func TestCollectionWrapperCountDocuments(t *testing.T) {
	t.Run("calls underlying collection CountDocuments", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "active"}

		mockColl.EXPECT().CountDocuments(mock.Anything, filter).Return(int64(42), nil)

		count, err := wrapper.CountDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), count)
	})
}

func TestCollectionWrapperDistinct(t *testing.T) {
	t.Run("calls underlying collection Distinct", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"status": "active"}
		expectedValues := []interface{}{"value1", "value2"}

		mockColl.EXPECT().Distinct(mock.Anything, "fieldName", filter).Return(expectedValues, nil)

		values, err := wrapper.Distinct(ctx, "fieldName", filter)
		assert.NoError(t, err)
		assert.Equal(t, expectedValues, values)
	})
}

func TestCollectionWrapperReplaceOne(t *testing.T) {
	t.Run("calls underlying collection ReplaceOne", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}
		replacement := map[string]interface{}{"name": "replaced"}
		expectedResult := &mongodriver.UpdateResult{ModifiedCount: 1}

		mockColl.EXPECT().ReplaceOne(mock.Anything, filter, replacement).Return(expectedResult, nil)

		result, err := wrapper.ReplaceOne(ctx, filter, replacement)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperBulkWrite(t *testing.T) {
	t.Run("calls underlying collection BulkWrite", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		models := []mongodriver.WriteModel{
			mongodriver.NewInsertOneModel().SetDocument(map[string]interface{}{"name": "test"}),
		}
		expectedResult := &mongodriver.BulkWriteResult{InsertedCount: 1}

		mockColl.EXPECT().BulkWrite(mock.Anything, models).Return(expectedResult, nil)

		result, err := wrapper.BulkWrite(ctx, models)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestCollectionWrapperIndexes(t *testing.T) {
	t.Run("calls underlying collection Indexes", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		mockColl.EXPECT().Indexes().Return(mongodriver.IndexView{})

		indexView := wrapper.Indexes()
		assert.NotNil(t, indexView)
	})
}

func TestCollectionWrapperDrop(t *testing.T) {
	t.Run("calls underlying collection Drop successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()

		mockColl.EXPECT().Drop(mock.Anything).Return(nil)

		err := wrapper.Drop(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error from underlying collection", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		ctx := context.Background()
		expectedErr := errors.New("drop error")

		mockColl.EXPECT().Drop(mock.Anything).Return(expectedErr)

		err := wrapper.Drop(ctx)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestCollectionWrapperName(t *testing.T) {
	t.Run("returns collection name", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		mockColl.EXPECT().Name().Return("test_collection")

		name := wrapper.Name()
		assert.Equal(t, "test_collection", name)
	})
}

func TestCollectionWrapperDatabase(t *testing.T) {
	t.Run("returns database", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		wrapper := newCollectionWrapper(mockColl)

		mockColl.EXPECT().Database().Return(nil)

		db := wrapper.Database()
		assert.Nil(t, db)
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	t.Run("middleware is called for each operation", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		callCount := 0
		trackingMw := func(ctx context.Context, next func(context.Context)) {
			callCount++
			next(ctx)
		}

		wrapper := newCollectionWrapper(mockColl, WithMiddleware(trackingMw))

		ctx := context.Background()
		filter := map[string]interface{}{"_id": "123"}

		mockColl.EXPECT().FindOne(mock.Anything, filter).Return(&mongodriver.SingleResult{})
		mockColl.EXPECT().DeleteOne(mock.Anything, filter).Return(&mongodriver.DeleteResult{}, nil)

		wrapper.FindOne(ctx, filter)
		wrapper.DeleteOne(ctx, filter)

		assert.Equal(t, 2, callCount)
	})

	t.Run("context is modified by middleware", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)

		type contextKey string
		key := contextKey("test-key")
		var capturedCtx context.Context

		contextMw := func(ctx context.Context, next func(context.Context)) {
			newCtx := context.WithValue(ctx, key, "test-value")
			next(newCtx)
		}

		wrapper := newCollectionWrapper(mockColl, WithMiddleware(contextMw))

		mockColl.EXPECT().FindOne(mock.Anything, mock.Anything).Run(func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) {
			capturedCtx = ctx
		}).Return(&mongodriver.SingleResult{})

		wrapper.FindOne(context.Background(), map[string]interface{}{})

		assert.Equal(t, "test-value", capturedCtx.Value(key))
	})

	t.Run("multiple middlewares are chained correctly", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		order := []string{}

		mw1 := func(ctx context.Context, next func(context.Context)) {
			order = append(order, "mw1-before")
			next(ctx)
			order = append(order, "mw1-after")
		}

		mw2 := func(ctx context.Context, next func(context.Context)) {
			order = append(order, "mw2-before")
			next(ctx)
			order = append(order, "mw2-after")
		}

		wrapper := newCollectionWrapper(mockColl, WithMiddleware(mw1), WithMiddleware(mw2))

		mockColl.EXPECT().FindOne(mock.Anything, mock.Anything).Run(func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) {
			order = append(order, "operation")
		}).Return(&mongodriver.SingleResult{})

		wrapper.FindOne(context.Background(), map[string]interface{}{})

		assert.Equal(t, []string{"mw1-before", "mw2-before", "operation", "mw2-after", "mw1-after"}, order)
	})
}

func TestWithTimeoutOption(t *testing.T) {
	t.Run("applies timeout to operations", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		timeout := 5 * time.Second

		var capturedCtx context.Context
		mockColl.EXPECT().FindOne(mock.Anything, mock.Anything).Run(func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) {
			capturedCtx = ctx
		}).Return(&mongodriver.SingleResult{})

		wrapper := newCollectionWrapper(mockColl, WithTimeout(timeout))
		wrapper.FindOne(context.Background(), map[string]interface{}{})

		deadline, ok := capturedCtx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(timeout), deadline, 100*time.Millisecond)
	})
}
