package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QueryOptions defines options for querying entities with filtering, pagination and sorting.
type QueryOptions struct {
	// Filter is the MongoDB filter criteria (BSON)
	Filter bson.D
	// Page is the page number (1-based)
	Page int
	// Size is the number of items per page
	Size int
	// Sort is the MongoDB sort criteria (BSON)
	// Example: bson.D{{"createdAt", -1}} for descending order
	Sort bson.D
}

// PageResult represents a paginated result.
type PageResult[Domain any] struct {
	// Items is the list of domain objects for the current page
	Items []*Domain
	// Total is the total number of items matching the filter
	Total int64
	// Page is the current page number (1-based)
	Page int
	// Size is the number of items per page
	Size int
	// TotalPages is the total number of pages
	TotalPages int
}

// EntityMapper defines the contract for converting between domain models and MongoDB entities.
// Each repository implementation must provide this mapper.
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

// GenericRepository provides common CRUD operations for MongoDB.
type GenericRepository[Domain any, Entity any] struct {
	coll   Collection
	mapper EntityMapper[Domain, Entity]
}

// NewGenericRepository creates a new generic repository.
// Returns error if collection or mapper is nil.
func NewGenericRepository[Domain any, Entity any](
	coll Collection,
	mapper EntityMapper[Domain, Entity],
) (*GenericRepository[Domain, Entity], error) {
	if coll == nil {
		return nil, fmt.Errorf("collection is required")
	}
	if mapper == nil {
		return nil, fmt.Errorf("mapper is required")
	}
	return &GenericRepository[Domain, Entity]{
		coll:   coll,
		mapper: mapper,
	}, nil
}

// Insert creates a new entity in MongoDB.
func (r *GenericRepository[Domain, Entity]) Insert(ctx context.Context, domain *Domain) error {
	entity := r.mapper.ToEntity(domain)

	_, err := r.coll.InsertOne(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to insert entity: %w", err)
	}

	return nil
}

// FindByID retrieves an entity by ID.
func (r *GenericRepository[Domain, Entity]) FindByID(ctx context.Context, id string) (*Domain, error) {
	result := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}})

	var entity Entity
	err := result.Decode(&entity)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, persistence.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to decode entity: %w", err)
	}

	return r.mapper.ToDomain(&entity), nil
}

// FindAll retrieves all entities.
func (r *GenericRepository[Domain, Entity]) FindAll(ctx context.Context) ([]*Domain, error) {
	cursor, err := r.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

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

// FindWithOptions retrieves entities with filtering, pagination and sorting.
func (r *GenericRepository[Domain, Entity]) FindWithOptions(
	ctx context.Context,
	opts QueryOptions,
) (*PageResult[Domain], error) {
	// Set default values
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Size < 1 {
		opts.Size = 10
	}
	if opts.Filter == nil {
		opts.Filter = bson.D{}
	}

	// Count total documents matching the filter
	total, err := r.coll.CountDocuments(ctx, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// Calculate pagination
	skip := int64((opts.Page - 1) * opts.Size)
	limit := int64(opts.Size)

	// Build find options
	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(limit)

	if opts.Sort != nil {
		findOpts.SetSort(opts.Sort)
	}

	// Execute query
	cursor, err := r.coll.Find(ctx, opts.Filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	// Decode results
	var entities []Entity
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("failed to decode entities: %w", err)
	}

	// Convert to domain objects
	domains := make([]*Domain, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.mapper.ToDomain(&entities[i]))
	}

	// Calculate total pages
	totalPages := int(total) / opts.Size
	if int(total)%opts.Size != 0 {
		totalPages++
	}

	return &PageResult[Domain]{
		Items:      domains,
		Total:      total,
		Page:       opts.Page,
		Size:       opts.Size,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing entity with optimistic locking and returns the updated domain object.
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
			// Return optimistic locking error
			return nil, persistence.ErrOptimisticLocking
		}
		return nil, fmt.Errorf("failed to update entity: %w", result.Err())
	}

	var updated Entity
	if err := result.Decode(&updated); err != nil {
		return nil, fmt.Errorf("failed to decode updated entity: %w", err)
	}

	return r.mapper.ToDomain(&updated), nil
}

// Delete hard deletes an entity by ID.
func (r *GenericRepository[Domain, Entity]) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// Exists checks if an entity with the given ID exists.
func (r *GenericRepository[Domain, Entity]) Exists(ctx context.Context, id string) (bool, error) {
	count, err := r.coll.CountDocuments(ctx, bson.D{{Key: "_id", Value: id}}, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}
	return count > 0, nil
}

// ExistsWithFilter checks if any entity matching the filter exists.
func (r *GenericRepository[Domain, Entity]) ExistsWithFilter(ctx context.Context, filter bson.D) (bool, error) {
	count, err := r.coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}
	return count > 0, nil
}

// UpsertIfNewer inserts or replaces an entity only if its version is greater than the existing one.
// This is useful for CQRS projections where events may arrive out of order.
// Returns true if the entity was inserted/updated, false if skipped due to version conflict.
func (r *GenericRepository[Domain, Entity]) UpsertIfNewer(ctx context.Context, domain *Domain) (bool, error) {
	entity := r.mapper.ToEntity(domain)

	filter := bson.D{
		{Key: "_id", Value: r.mapper.GetID(entity)},
		{Key: "version", Value: bson.M{"$lt": r.mapper.GetVersion(entity)}},
	}

	opts := options.Replace().SetUpsert(true)
	result, err := r.coll.ReplaceOne(ctx, filter, entity, opts)
	if err != nil {
		return false, fmt.Errorf("failed to upsert entity: %w", err)
	}

	// If no match and no upsert, it means version conflict (existing doc has >= version)
	updated := result.MatchedCount > 0 || result.UpsertedCount > 0
	return updated, nil
}
