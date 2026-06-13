package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Status represents the lifecycle state of a tenant in the local registry.
type Status string

const (
	// StatusActive indicates a tenant is active and operational.
	StatusActive Status = "active"
	// StatusPendingDeletion indicates a tenant is marked for future deletion.
	StatusPendingDeletion Status = "pending_deletion"
)

// Record represents a tenant entry in the local registry collection.
type Record struct {
	Slug        string     `bson:"_id"`
	Status      Status     `bson:"status"`
	DeleteAfter *time.Time `bson:"delete_after,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
	ModifiedAt  time.Time  `bson:"modified_at"`
}

// repository provides access to the local tenant registry stored in the shared (admin) database.
// Used for:
//   - Fallback tenant list when tenant-service API is unavailable
//   - Deferred cleanup: marking tenants for deletion and querying pending deletions
type repository interface {
	// Upsert creates or updates a tenant record with status=active.
	Upsert(ctx context.Context, slug string) error

	// MarkForDeletion sets a tenant's status to pending_deletion with a future delete_after timestamp.
	MarkForDeletion(ctx context.Context, slug string, deleteAfter time.Time) error

	// FindPendingDeletion returns all tenants where status=pending_deletion AND delete_after <= now.
	FindPendingDeletion(ctx context.Context) ([]Record, error)

	// FindActive returns all tenants with status=active.
	FindActive(ctx context.Context) ([]Record, error)

	// Remove permanently deletes a tenant record from the registry.
	Remove(ctx context.Context, slug string) error
}

type mongoRepository struct {
	coll *mongodriver.Collection
}

// newMongoRepository creates a repository backed by MongoDB.
func newMongoRepository(admin mongo.Admin) repository {
	return &mongoRepository{
		coll: admin.GetDatabase().Collection("tenants"),
	}
}

func (r *mongoRepository) Upsert(ctx context.Context, slug string) error {
	now := time.Now()
	filter := bson.D{{Key: "_id", Value: slug}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusActive},
			{Key: "delete_after", Value: nil},
			{Key: "modified_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: now},
		}},
	}

	opts := options.UpdateOne().SetUpsert(true)
	if _, err := r.coll.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("failed to upsert tenant %q: %w", slug, err)
	}
	return nil
}

func (r *mongoRepository) MarkForDeletion(ctx context.Context, slug string, deleteAfter time.Time) error {
	now := time.Now()
	filter := bson.D{{Key: "_id", Value: slug}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusPendingDeletion},
			{Key: "delete_after", Value: deleteAfter},
			{Key: "modified_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "created_at", Value: now},
		}},
	}

	opts := options.UpdateOne().SetUpsert(true)
	if _, err := r.coll.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("failed to mark tenant %q for deletion: %w", slug, err)
	}
	return nil
}

func (r *mongoRepository) FindPendingDeletion(ctx context.Context) ([]Record, error) {
	filter := bson.D{
		{Key: "status", Value: StatusPendingDeletion},
		{Key: "delete_after", Value: bson.D{{Key: "$lte", Value: time.Now()}}},
	}

	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending deletion tenants: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }() //nolint:errcheck // Best effort cleanup

	var records []Record
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode tenant records: %w", err)
	}
	return records, nil
}

func (r *mongoRepository) FindActive(ctx context.Context) ([]Record, error) {
	filter := bson.D{{Key: "status", Value: StatusActive}}

	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find active tenants: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }() //nolint:errcheck // Best effort cleanup

	var records []Record
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode tenant records: %w", err)
	}
	return records, nil
}

func (r *mongoRepository) Remove(ctx context.Context, slug string) error {
	filter := bson.D{{Key: "_id", Value: slug}}
	if _, err := r.coll.DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("failed to remove tenant %q from registry: %w", slug, err)
	}
	return nil
}
