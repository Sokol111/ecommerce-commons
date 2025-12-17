package mongo

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence"
	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo/mongomock"
)

// Test domain and entity types
type testDomain struct {
	ID      string
	Name    string
	Version int
}

type testEntity struct {
	ID      string `bson:"_id"`
	Name    string `bson:"name"`
	Version int    `bson:"version"`
}

// mockMapper is a mock implementation of EntityMapper
type mockMapper struct {
	mock.Mock
}

func (m *mockMapper) ToEntity(domain *testDomain) *testEntity {
	args := m.Called(domain)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*testEntity)
}

func (m *mockMapper) ToDomain(entity *testEntity) *testDomain {
	args := m.Called(entity)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*testDomain)
}

func (m *mockMapper) GetID(entity *testEntity) string {
	args := m.Called(entity)
	return args.String(0)
}

func (m *mockMapper) GetVersion(entity *testEntity) int {
	args := m.Called(entity)
	return args.Int(0)
}

func (m *mockMapper) SetVersion(entity *testEntity, version int) {
	m.Called(entity, version)
}

// TestNewGenericRepository tests the constructor
func TestNewGenericRepository(t *testing.T) {
	t.Run("creates repository successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		repo, err := NewGenericRepository[testDomain, testEntity](mockColl, mapper)

		assert.NoError(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("returns error when collection is nil", func(t *testing.T) {
		mapper := &mockMapper{}

		repo, err := NewGenericRepository[testDomain, testEntity](nil, mapper)

		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "collection is required")
	})

	t.Run("returns error when mapper is nil", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)

		repo, err := NewGenericRepository[testDomain, testEntity](mockColl, nil)

		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "mapper is required")
	})
}

// TestGenericRepository_Save tests the Save method
func TestGenericRepository_Save(t *testing.T) {
	t.Run("saves entity successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "test", Version: 1}
		entity := &testEntity{ID: "123", Name: "test", Version: 1}

		mapper.On("ToEntity", domain).Return(entity)
		mockColl.EXPECT().InsertOne(mock.Anything, entity).Return(&mongodriver.InsertOneResult{InsertedID: "123"}, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		err := repo.Insert(context.Background(), domain)

		assert.NoError(t, err)
		mapper.AssertExpectations(t)
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "test", Version: 1}
		entity := &testEntity{ID: "123", Name: "test", Version: 1}
		expectedErr := errors.New("insert failed")

		mapper.On("ToEntity", domain).Return(entity)
		mockColl.EXPECT().InsertOne(mock.Anything, entity).Return(nil, expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		err := repo.Insert(context.Background(), domain)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert entity")
		mapper.AssertExpectations(t)
	})
}

// TestGenericRepository_FindByID tests the FindByID method
func TestGenericRepository_FindByID(t *testing.T) {
	t.Run("finds entity by ID successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		entity := testEntity{ID: "123", Name: "test", Version: 1}
		domain := &testDomain{ID: "123", Name: "test", Version: 1}

		mockColl.EXPECT().FindOne(mock.Anything, bson.D{{Key: "_id", Value: "123"}}).
			Run(func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) {
				// Verify filter
			}).
			Return(mongodriver.NewSingleResultFromDocument(entity, nil, nil))

		mapper.On("ToDomain", mock.AnythingOfType("*mongo.testEntity")).Return(domain)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.FindByID(context.Background(), "123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "123", result.ID)
		mapper.AssertExpectations(t)
	})

	t.Run("returns ErrEntityNotFound when entity not found", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		// Create a SingleResult that returns ErrNoDocuments
		singleResult := mongodriver.NewSingleResultFromDocument(bson.D{}, mongodriver.ErrNoDocuments, nil)

		mockColl.EXPECT().FindOne(mock.Anything, bson.D{{Key: "_id", Value: "not-found"}}).
			Return(singleResult)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.FindByID(context.Background(), "not-found")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, persistence.ErrEntityNotFound)
	})
}

// TestGenericRepository_FindAll tests the FindAll method
func TestGenericRepository_FindAll(t *testing.T) {
	t.Run("finds all entities successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		entities := []testEntity{
			{ID: "1", Name: "test1", Version: 1},
			{ID: "2", Name: "test2", Version: 1},
		}

		cursor, _ := mongodriver.NewCursorFromDocuments(toInterfaceSlice(entities), nil, nil)
		mockColl.EXPECT().Find(mock.Anything, bson.D{}).Return(cursor, nil)

		mapper.On("ToDomain", mock.AnythingOfType("*mongo.testEntity")).Return(&testDomain{ID: "1", Name: "test1", Version: 1}).Once()
		mapper.On("ToDomain", mock.AnythingOfType("*mongo.testEntity")).Return(&testDomain{ID: "2", Name: "test2", Version: 1}).Once()

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		results, err := repo.FindAll(context.Background())

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		mapper.AssertExpectations(t)
	})

	t.Run("returns empty slice when no entities", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		cursor, _ := mongodriver.NewCursorFromDocuments([]interface{}{}, nil, nil)
		mockColl.EXPECT().Find(mock.Anything, bson.D{}).Return(cursor, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		results, err := repo.FindAll(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		expectedErr := errors.New("query failed")
		mockColl.EXPECT().Find(mock.Anything, bson.D{}).Return(nil, expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		results, err := repo.FindAll(context.Background())

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "failed to query entities")
	})
}

// TestGenericRepository_FindWithOptions tests the FindWithOptions method
func TestGenericRepository_FindWithOptions(t *testing.T) {
	t.Run("finds with pagination successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		entities := []testEntity{
			{ID: "1", Name: "test1", Version: 1},
		}

		opts := QueryOptions{
			Page: 1,
			Size: 10,
		}

		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{}).Return(int64(25), nil)

		cursor, _ := mongodriver.NewCursorFromDocuments(toInterfaceSlice(entities), nil, nil)
		mockColl.EXPECT().Find(mock.Anything, bson.D{}, mock.Anything).Return(cursor, nil)

		mapper.On("ToDomain", mock.AnythingOfType("*mongo.testEntity")).Return(&testDomain{ID: "1", Name: "test1", Version: 1})

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.FindWithOptions(context.Background(), opts)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Size)
		assert.Equal(t, 3, result.TotalPages) // 25 / 10 = 2.5 -> 3
		mapper.AssertExpectations(t)
	})

	t.Run("uses default values for invalid page and size", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		opts := QueryOptions{
			Page: 0,  // invalid, should default to 1
			Size: -1, // invalid, should default to 10
		}

		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{}).Return(int64(5), nil)

		cursor, _ := mongodriver.NewCursorFromDocuments([]interface{}{}, nil, nil)
		mockColl.EXPECT().Find(mock.Anything, bson.D{}, mock.Anything).Return(cursor, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.FindWithOptions(context.Background(), opts)

		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Size)
	})

	t.Run("applies filter correctly", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		filter := bson.D{{Key: "status", Value: "active"}}
		opts := QueryOptions{
			Filter: filter,
			Page:   1,
			Size:   10,
		}

		mockColl.EXPECT().CountDocuments(mock.Anything, filter).Return(int64(0), nil)

		cursor, _ := mongodriver.NewCursorFromDocuments([]interface{}{}, nil, nil)
		mockColl.EXPECT().Find(mock.Anything, filter, mock.Anything).Return(cursor, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		_, err := repo.FindWithOptions(context.Background(), opts)

		assert.NoError(t, err)
	})

	t.Run("returns error when count fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		expectedErr := errors.New("count failed")
		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{}).Return(int64(0), expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.FindWithOptions(context.Background(), QueryOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to count entities")
	})

	t.Run("calculates total pages correctly", func(t *testing.T) {
		testCases := []struct {
			total         int64
			size          int
			expectedPages int
		}{
			{total: 0, size: 10, expectedPages: 0},
			{total: 1, size: 10, expectedPages: 1},
			{total: 10, size: 10, expectedPages: 1},
			{total: 11, size: 10, expectedPages: 2},
			{total: 25, size: 10, expectedPages: 3},
			{total: 100, size: 10, expectedPages: 10},
		}

		for _, tc := range testCases {
			mockColl := mongomock.NewMockCollection(t)
			mapper := &mockMapper{}

			mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{}).Return(tc.total, nil)

			cursor, _ := mongodriver.NewCursorFromDocuments([]interface{}{}, nil, nil)
			mockColl.EXPECT().Find(mock.Anything, bson.D{}, mock.Anything).Return(cursor, nil)

			repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
			result, err := repo.FindWithOptions(context.Background(), QueryOptions{Size: tc.size})

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedPages, result.TotalPages, "total=%d, size=%d", tc.total, tc.size)
		}
	})
}

// TestGenericRepository_Update tests the Update method
func TestGenericRepository_Update(t *testing.T) {
	t.Run("updates entity successfully with optimistic locking", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "updated", Version: 1}
		entity := &testEntity{ID: "123", Name: "updated", Version: 1}
		updatedEntity := testEntity{ID: "123", Name: "updated", Version: 2}
		updatedDomain := &testDomain{ID: "123", Name: "updated", Version: 2}

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetVersion", entity).Return(1)
		mapper.On("SetVersion", entity, 2).Return()
		mapper.On("GetID", entity).Return("123")
		mapper.On("ToDomain", mock.AnythingOfType("*mongo.testEntity")).Return(updatedDomain)

		expectedFilter := bson.D{
			{Key: "_id", Value: "123"},
			{Key: "version", Value: 1},
		}

		mockColl.EXPECT().FindOneAndReplace(mock.Anything, expectedFilter, entity, mock.Anything).
			Return(mongodriver.NewSingleResultFromDocument(updatedEntity, nil, nil))

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.Update(context.Background(), domain)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Version)
		mapper.AssertExpectations(t)
	})

	t.Run("returns ErrOptimisticLocking when version mismatch", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "updated", Version: 1}
		entity := &testEntity{ID: "123", Name: "updated", Version: 1}

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetVersion", entity).Return(1)
		mapper.On("SetVersion", entity, 2).Return()
		mapper.On("GetID", entity).Return("123")

		// Create a SingleResult that returns ErrNoDocuments (version mismatch or not found)
		singleResult := mongodriver.NewSingleResultFromDocument(bson.D{}, mongodriver.ErrNoDocuments, nil)

		mockColl.EXPECT().FindOneAndReplace(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(singleResult)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.Update(context.Background(), domain)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, persistence.ErrOptimisticLocking)
		mapper.AssertExpectations(t)
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "updated", Version: 1}
		entity := &testEntity{ID: "123", Name: "updated", Version: 1}
		expectedErr := errors.New("update failed")

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetVersion", entity).Return(1)
		mapper.On("SetVersion", entity, 2).Return()
		mapper.On("GetID", entity).Return("123")

		mockColl.EXPECT().FindOneAndReplace(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(mongodriver.NewSingleResultFromDocument(nil, expectedErr, nil))

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		result, err := repo.Update(context.Background(), domain)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to update entity")
		mapper.AssertExpectations(t)
	})
}

// TestGenericRepository_Delete tests the Delete method
func TestGenericRepository_Delete(t *testing.T) {
	t.Run("deletes entity successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		mockColl.EXPECT().DeleteOne(mock.Anything, bson.D{{Key: "_id", Value: "123"}}).
			Return(&mongodriver.DeleteResult{DeletedCount: 1}, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		err := repo.Delete(context.Background(), "123")

		assert.NoError(t, err)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		expectedErr := errors.New("delete failed")
		mockColl.EXPECT().DeleteOne(mock.Anything, bson.D{{Key: "_id", Value: "123"}}).
			Return(nil, expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		err := repo.Delete(context.Background(), "123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete entity")
	})
}

// TestGenericRepository_Exists tests the Exists method
func TestGenericRepository_Exists(t *testing.T) {
	t.Run("returns true when entity exists", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{{Key: "_id", Value: "123"}}, mock.Anything).
			Return(int64(1), nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.Exists(context.Background(), "123")

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false when entity does not exist", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{{Key: "_id", Value: "not-found"}}, mock.Anything).
			Return(int64(0), nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.Exists(context.Background(), "not-found")

		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns error when count fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		expectedErr := errors.New("count failed")
		mockColl.EXPECT().CountDocuments(mock.Anything, bson.D{{Key: "_id", Value: "123"}}, mock.Anything).
			Return(int64(0), expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.Exists(context.Background(), "123")

		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "failed to check entity existence")
	})
}

// TestGenericRepository_ExistsWithFilter tests the ExistsWithFilter method
func TestGenericRepository_ExistsWithFilter(t *testing.T) {
	t.Run("returns true when entities match filter", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		filter := bson.D{{Key: "status", Value: "active"}}
		mockColl.EXPECT().CountDocuments(mock.Anything, filter, mock.Anything).
			Return(int64(1), nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.ExistsWithFilter(context.Background(), filter)

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false when no entities match filter", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		filter := bson.D{{Key: "status", Value: "inactive"}}
		mockColl.EXPECT().CountDocuments(mock.Anything, filter, mock.Anything).
			Return(int64(0), nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.ExistsWithFilter(context.Background(), filter)

		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns error when count fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		filter := bson.D{{Key: "status", Value: "active"}}
		expectedErr := errors.New("count failed")
		mockColl.EXPECT().CountDocuments(mock.Anything, filter, mock.Anything).
			Return(int64(0), expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		exists, err := repo.ExistsWithFilter(context.Background(), filter)

		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "failed to check entity existence")
	})
}

// TestGenericRepository_UpsertIfNewer tests the UpsertIfNewer method
func TestGenericRepository_UpsertIfNewer(t *testing.T) {
	t.Run("inserts new entity successfully", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "test", Version: 1}
		entity := &testEntity{ID: "123", Name: "test", Version: 1}

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetID", entity).Return("123")
		mapper.On("GetVersion", entity).Return(1)

		mockColl.EXPECT().ReplaceOne(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(&mongodriver.UpdateResult{UpsertedCount: 1}, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		updated, err := repo.UpsertIfNewer(context.Background(), domain)

		assert.NoError(t, err)
		assert.True(t, updated)
		mapper.AssertExpectations(t)
	})

	t.Run("updates existing entity with older version", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "updated", Version: 2}
		entity := &testEntity{ID: "123", Name: "updated", Version: 2}

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetID", entity).Return("123")
		mapper.On("GetVersion", entity).Return(2)

		mockColl.EXPECT().ReplaceOne(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(&mongodriver.UpdateResult{MatchedCount: 1}, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		updated, err := repo.UpsertIfNewer(context.Background(), domain)

		assert.NoError(t, err)
		assert.True(t, updated)
		mapper.AssertExpectations(t)
	})

	t.Run("skips update when existing version is newer or equal", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "old", Version: 1}
		entity := &testEntity{ID: "123", Name: "old", Version: 1}

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetID", entity).Return("123")
		mapper.On("GetVersion", entity).Return(1)

		// No match and no upsert means version conflict
		mockColl.EXPECT().ReplaceOne(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(&mongodriver.UpdateResult{MatchedCount: 0, UpsertedCount: 0}, nil)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		updated, err := repo.UpsertIfNewer(context.Background(), domain)

		assert.NoError(t, err)
		assert.False(t, updated)
		mapper.AssertExpectations(t)
	})

	t.Run("returns error when replace fails", func(t *testing.T) {
		mockColl := mongomock.NewMockCollection(t)
		mapper := &mockMapper{}

		domain := &testDomain{ID: "123", Name: "test", Version: 1}
		entity := &testEntity{ID: "123", Name: "test", Version: 1}
		expectedErr := errors.New("replace failed")

		mapper.On("ToEntity", domain).Return(entity)
		mapper.On("GetID", entity).Return("123")
		mapper.On("GetVersion", entity).Return(1)

		mockColl.EXPECT().ReplaceOne(mock.Anything, mock.Anything, entity, mock.Anything).
			Return(nil, expectedErr)

		repo, _ := NewGenericRepository[testDomain, testEntity](mockColl, mapper)
		updated, err := repo.UpsertIfNewer(context.Background(), domain)

		assert.Error(t, err)
		assert.False(t, updated)
		assert.Contains(t, err.Error(), "failed to upsert entity")
		mapper.AssertExpectations(t)
	})
}

// Helper function to convert slice to interface slice for cursor
func toInterfaceSlice[T any](slice []T) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
