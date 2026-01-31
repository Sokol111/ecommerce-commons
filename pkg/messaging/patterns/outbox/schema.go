package outbox

import (
	"context"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	collectionName = "outbox"

	// Index names.
	idxCreatedAtTTL          = "outbox_createdAt_ttl"
	idxStatusNextAttemptLock = "outbox_status_nextAttemptAfter_lockExpiresAt"

	// TTL for outbox documents (5 days).
	ttlSeconds = 5 * 24 * 60 * 60 // 432000 seconds
)

// EnsureIndexes creates required indexes for outbox collection.
// This is idempotent - safe to call multiple times.
func EnsureIndexes(ctx context.Context, m mongo.Mongo) error {
	coll := m.GetCollection(collectionName)

	indexes := []mongodriver.IndexModel{
		{
			Keys: bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().
				SetName(idxCreatedAtTTL).
				SetExpireAfterSeconds(ttlSeconds),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "nextAttemptAfter", Value: 1},
				{Key: "lockExpiresAt", Value: 1},
			},
			Options: options.Index().
				SetName(idxStatusNextAttemptLock),
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
