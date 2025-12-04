package mongo

import (
	"context"
	"errors"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo/mongomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// mockMongoAdmin is a mock for MongoAdmin interface
type mockMongoAdmin struct {
	mock.Mock
}

func (m *mockMongoAdmin) GetCollection(collection string) Collection {
	args := m.Called(collection)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Collection)
}

func (m *mockMongoAdmin) GetCollectionWithOptions(collection string, opts ...WrapperOption) Collection {
	args := m.Called(collection, opts)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Collection)
}

func (m *mockMongoAdmin) GetDatabase() *mongodriver.Database {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mongodriver.Database)
}

func (m *mockMongoAdmin) StartSession(ctx context.Context) (Session, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Session), args.Error(1)
}

func newMockMongoAdmin(t *testing.T) *mockMongoAdmin {
	m := &mockMongoAdmin{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func TestTxManagerWithTransaction(t *testing.T) {
	t.Run("executes transaction successfully", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()
		expectedResult := "transaction result"

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(expectedResult, nil)
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return expectedResult, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("returns error when start session fails", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		log := zap.NewNop()

		ctx := context.Background()
		expectedErr := errors.New("session start error")

		mockAdmin.On("StartSession", ctx).Return(nil, expectedErr)

		txManager := newTxManager(mockAdmin, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return "should not reach", nil
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to start session")
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("returns error when transaction fails", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()
		expectedErr := errors.New("transaction error")

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, expectedErr)
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, nil
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction failed")
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("session is always closed after successful transaction", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return("result", nil)
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return "result", nil
		})

		// Expectations are automatically verified by cleanup
	})

	t.Run("session is always closed after failed transaction", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, errors.New("some error"))
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, errors.New("some error")
		})

		// Expectations are automatically verified by cleanup
	})

	t.Run("user function is not called when session fails", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		log := zap.NewNop()

		ctx := context.Background()
		functionCalled := false

		mockAdmin.On("StartSession", ctx).Return(nil, errors.New("session error"))

		txManager := newTxManager(mockAdmin, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			functionCalled = true
			return nil, nil
		})

		assert.False(t, functionCalled, "user function should not be called when session start fails")
	})

	t.Run("returns custom result type", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()

		type customResult struct {
			ID   string
			Name string
		}
		expectedResult := customResult{ID: "123", Name: "test"}

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(expectedResult, nil)
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return expectedResult, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("returns nil result when user function returns nil", func(t *testing.T) {
		mockAdmin := newMockMongoAdmin(t)
		mockSess := mongomock.NewMockSession(t)
		log := zap.NewNop()

		ctx := context.Background()

		mockAdmin.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.EXPECT().WithTransaction(ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, nil)
		mockSess.EXPECT().EndSession(ctx).Return()

		txManager := newTxManager(mockAdmin, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, nil
		})

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
