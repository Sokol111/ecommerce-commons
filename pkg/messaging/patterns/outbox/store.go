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

type Store interface {

	// can return errEntityNotFound
	FetchAndLock(ctx context.Context) (*outboxEntity, error)

	Create(ctx context.Context, payload []byte, id string, key string, topic string, headers map[string]string) (*outboxEntity, error)

	UpdateAsSentByIds(ctx context.Context, ids []string) error
}

type store struct {
	coll mongo.Collection
}

func newStore(m mongo.Mongo) Store {
	return &store{
		coll: m.GetCollectionWrapper("outbox"),
	}
}

func (r *store) FetchAndLock(ctx context.Context) (*outboxEntity, error) {
	var entity outboxEntity

	opts := options.FindOneAndUpdate().SetSort(bson.D{
		{Key: "lockExpiresAt", Value: 1},
		{Key: "createdAt", Value: 1},
	}).SetReturnDocument(options.After)

	filter := bson.M{
		"$and": []bson.M{
			{"lockExpiresAt": bson.M{"$lt": time.Now().UTC()}},
			{"status": StatusProcessing},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"lockExpiresAt": time.Now().Add(30 * time.Second),
		},
		"$inc": bson.M{
			"attemptsToSend": 1,
		},
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

func (r *store) Create(ctx context.Context, payload []byte, id string, key string, topic string, headers map[string]string) (*outboxEntity, error) {
	entity := outboxEntity{
		ID:             id,
		Payload:        payload,
		Key:            key,
		Topic:          topic,
		Headers:        headers,
		CreatedAt:      time.Now().UTC(),
		Status:         StatusProcessing,
		LockExpiresAt:  time.Now().Add(10 * time.Second),
		AttemptsToSend: 0,
	}
	_, err := r.coll.InsertOne(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to insert outbox entity: %w", err)
	}
	return &entity, nil
}

func (r *store) UpdateAsSentByIds(ctx context.Context, ids []string) error {
	_, err := r.coll.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{
			"$set": bson.M{
				"status": StatusSent,
				"sentAt": time.Now().UTC(),
			},
			"$unset": bson.M{"lockExpiresAt": ""},
			"$inc":   bson.M{"confirmations": 1},
		})
	if err != nil {
		return fmt.Errorf("failed to update outbox messages: %w", err)
	}
	return nil
}
