package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EntityMapper defines the contract for converting between domain models and MongoDB entities
// Each repository implementation must provide this mapper
type EntityMapper[Domain any, Entity any] interface {
	// ToEntity converts domain model to MongoDB entity
	ToEntity(domain *Domain) *Entity

	// ToDomain converts MongoDB entity to domain model
	ToDomain(entity *Entity) *Domain

	// GetID extracts ID from entity (for queries)
	GetID(entity *Entity) string

	// GetVersion extracts version from entity (for optimistic locking)
	GetVersion(entity *Entity) int

	// SetVersion sets version on entity (for optimistic locking)
	SetVersion(entity *Entity, version int)
}

// GenericRepository provides common CRUD operations for MongoDB
type GenericRepository[Domain any, Entity any] struct {
	coll              Collection
	mapper            EntityMapper[Domain, Entity]
	notFoundErr       error // Domain-specific "not found" error
	optimisticLockErr error // Domain-specific "optimistic locking" error
}

// NewGenericRepository creates a new generic repository
func NewGenericRepository[Domain any, Entity any](
	coll Collection,
	mapper EntityMapper[Domain, Entity],
	notFoundErr error,
	optimisticLockErr error,
) *GenericRepository[Domain, Entity] {
	return &GenericRepository[Domain, Entity]{
		coll:              coll,
		mapper:            mapper,
		notFoundErr:       notFoundErr,
		optimisticLockErr: optimisticLockErr,
	}
}

// Save creates a new entity in MongoDB
func (r *GenericRepository[Domain, Entity]) Save(ctx context.Context, domain *Domain) error {
	entity := r.mapper.ToEntity(domain)

	_, err := r.coll.InsertOne(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to insert entity: %w", err)
	}

	return nil
}

// FindByID retrieves an entity by ID
func (r *GenericRepository[Domain, Entity]) FindByID(ctx context.Context, id string) (*Domain, error) {
	result := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}})

	var entity Entity
	err := result.Decode(&entity)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, r.notFoundErr
		}
		return nil, fmt.Errorf("failed to decode entity: %w", err)
	}

	return r.mapper.ToDomain(&entity), nil
}

// FindAll retrieves all entities
func (r *GenericRepository[Domain, Entity]) FindAll(ctx context.Context) ([]*Domain, error) {
	cursor, err := r.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer cursor.Close(ctx)

	var entities []Entity
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("failed to decode entities: %w", err)
	}

	domains := make([]*Domain, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.mapper.ToDomain(&entities[i]))
	}

	return domains, nil
}

// Update updates an existing entity with optimistic locking and returns the updated domain object
func (r *GenericRepository[Domain, Entity]) Update(ctx context.Context, domain *Domain) (*Domain, error) {
	entity := r.mapper.ToEntity(domain)

	// Get current version for optimistic locking
	currentVersion := r.mapper.GetVersion(entity)

	// Increment version for optimistic locking
	newVersion := currentVersion + 1
	r.mapper.SetVersion(entity, newVersion)

	opts := options.FindOneAndReplace().SetReturnDocument(options.After)
	result := r.coll.FindOneAndReplace(
		ctx,
		bson.D{
			{Key: "_id", Value: r.mapper.GetID(entity)},
			{Key: "version", Value: currentVersion}, // Match old version
		},
		entity,
		opts,
	)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongodriver.ErrNoDocuments) {
			// This could be either not found or version mismatch
			// Return domain-specific optimistic locking error
			return nil, r.optimisticLockErr
		}
		return nil, fmt.Errorf("failed to update entity: %w", result.Err())
	}

	var updated Entity
	if err := result.Decode(&updated); err != nil {
		return nil, fmt.Errorf("failed to decode updated entity: %w", err)
	}

	return r.mapper.ToDomain(&updated), nil
}
