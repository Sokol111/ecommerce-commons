package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/persistence/mongo"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var errEntityNotFound = errors.New("entity not found in database")

type repository interface {

	// can return errEntityNotFound
	FetchAndLock(ctx context.Context) (*outboxEntity, error)

	Create(ctx context.Context, payload []byte, id string, key string, topic string, headers map[string]string) (*outboxEntity, error)

	UpdateAsSentByIds(ctx context.Context, ids []string) error
}

type outboxRepository struct {
	coll             mongo.Collection
	maxBackoffMillis int64
}

func newOutboxRepository(m mongo.Mongo, config *Config) repository {
	return &outboxRepository{
		coll:             m.GetCollection("outbox"),
		maxBackoffMillis: config.MaxBackoff.Milliseconds(),
	}
}

func (r *outboxRepository) FetchAndLock(ctx context.Context) (*outboxEntity, error) {
	var entity outboxEntity

	now := time.Now().UTC()

	opts := options.FindOneAndUpdate().SetSort(bson.D{
		{Key: "nextAttemptAfter", Value: 1},
		{Key: "createdAt", Value: 1},
	}).SetReturnDocument(options.After)

	filter := bson.M{
		"$and": []bson.M{
			{"nextAttemptAfter": bson.M{"$lt": now}},
			{"lockExpiresAt": bson.M{"$lt": now}},
			{"status": StatusProcessing},
		},
	}

	// Aggregation pipeline to calculate exponential backoff:
	// delay = min(baseDelay * 2^attempts, maxDelay)
	// baseDelay = 30s, maxDelay = config.MaxBackoff
	update := mongodriver.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			"lockExpiresAt":  now.Add(30 * time.Second),
			"attemptsToSend": bson.M{"$add": []any{"$attemptsToSend", 1}},
			"nextAttemptAfter": bson.M{
				"$add": []any{
					now,
					bson.M{"$min": []any{
						bson.M{"$multiply": []any{
							30000, // 30 seconds in milliseconds
							bson.M{"$pow": []any{2, "$attemptsToSend"}},
						}},
						r.maxBackoffMillis,
					}},
				},
			},
		}}},
	}

	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&entity)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, fmt.Errorf("failed to fetch outbox entity: %w", errEntityNotFound)
		}
		return nil, fmt.Errorf("failed to fetch outbox entity: %w", err)
	}

	return &entity, nil
}

func (r *outboxRepository) Create(ctx context.Context, payload []byte, id string, key string, topic string, headers map[string]string) (*outboxEntity, error) {
	now := time.Now().UTC()
	entity := outboxEntity{
		ID:               id,
		Payload:          payload,
		Key:              key,
		Topic:            topic,
		Headers:          headers,
		CreatedAt:        now,
		Status:           StatusProcessing,
		LockExpiresAt:    now.Add(10 * time.Second),
		NextAttemptAfter: now.Add(10 * time.Second),
		AttemptsToSend:   0,
	}
	_, err := r.coll.InsertOne(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to insert outbox entity: %w", err)
	}
	return &entity, nil
}

func (r *outboxRepository) UpdateAsSentByIds(ctx context.Context, ids []string) error {
	_, err := r.coll.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{
			"$set": bson.M{
				"status": StatusSent,
				"sentAt": time.Now().UTC(),
			},
			"$unset": bson.M{
				"lockExpiresAt":    "",
				"nextAttemptAfter": "",
			},
			"$inc": bson.M{"confirmations": 1},
		})
	if err != nil {
		return fmt.Errorf("failed to update outbox messages: %w", err)
	}
	return nil
}
