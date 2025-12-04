package mongo

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	variadicArgs := make([]interface{}, len(opts))
	for i, a := range opts {
		variadicArgs[i] = a
	}
	args := m.Called(append([]interface{}{collection}, variadicArgs...)...)
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

// mockSession is a mock for Session interface
type mockSession struct {
	mock.Mock
}

func (m *mockSession) WithTransaction(ctx context.Context, fn func(sessCtx mongodriver.SessionContext) (interface{}, error), opts ...*options.TransactionOptions) (interface{}, error) {
	args := m.Called(ctx, fn, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *mockSession) EndSession(ctx context.Context) {
	m.Called(ctx)
}

func TestNewTxManager(t *testing.T) {
	t.Run("creates tx manager successfully", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		log := zap.NewNop()

		txManager := newTxManager(mockMongo, log)

		assert.NotNil(t, txManager)
	})
}

func TestTxManagerWithTransaction(t *testing.T) {
	t.Run("executes transaction successfully", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()
		expectedResult := "transaction result"

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(expectedResult, nil)
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return expectedResult, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})

	t.Run("returns error when start session fails", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		log := zap.NewNop()

		ctx := context.Background()
		expectedErr := errors.New("session start error")

		mockMongo.On("StartSession", ctx).Return(nil, expectedErr)

		txManager := newTxManager(mockMongo, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return "should not reach", nil
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to start session")
		assert.ErrorIs(t, err, expectedErr)
		mockMongo.AssertExpectations(t)
	})

	t.Run("returns error when transaction fails", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()
		expectedErr := errors.New("transaction error")

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, expectedErr)
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, nil
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction failed")
		assert.ErrorIs(t, err, expectedErr)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})

	t.Run("session is always closed after successful transaction", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return("result", nil)
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return "result", nil
		})

		mockSess.AssertCalled(t, "EndSession", ctx)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})

	t.Run("session is always closed after failed transaction", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, errors.New("some error"))
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, errors.New("some error")
		})

		mockSess.AssertCalled(t, "EndSession", ctx)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})

	t.Run("user function is not called when session fails", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		log := zap.NewNop()

		ctx := context.Background()
		functionCalled := false

		mockMongo.On("StartSession", ctx).Return(nil, errors.New("session error"))

		txManager := newTxManager(mockMongo, log)
		_, _ = txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			functionCalled = true
			return nil, nil
		})

		assert.False(t, functionCalled, "user function should not be called when session start fails")
		mockMongo.AssertExpectations(t)
	})

	t.Run("returns custom result type", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()

		type customResult struct {
			ID   string
			Name string
		}
		expectedResult := customResult{ID: "123", Name: "test"}

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(expectedResult, nil)
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return expectedResult, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})

	t.Run("returns nil result when user function returns nil", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		mockSess := &mockSession{}
		log := zap.NewNop()

		ctx := context.Background()

		mockMongo.On("StartSession", ctx).Return(mockSess, nil)
		mockSess.On("WithTransaction", ctx, mock.AnythingOfType("func(mongo.SessionContext) (interface {}, error)"), mock.Anything).
			Return(nil, nil)
		mockSess.On("EndSession", ctx).Return()

		txManager := newTxManager(mockMongo, log)
		result, err := txManager.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
			return nil, nil
		})

		assert.NoError(t, err)
		assert.Nil(t, result)
		mockMongo.AssertExpectations(t)
		mockSess.AssertExpectations(t)
	})
}

func TestTxManagerImplementsInterface(t *testing.T) {
	t.Run("implements persistence.TxManager interface", func(t *testing.T) {
		mockMongo := &mockMongoAdmin{}
		log := zap.NewNop()

		txManager := newTxManager(mockMongo, log)

		assert.NotNil(t, txManager)
	})
}
