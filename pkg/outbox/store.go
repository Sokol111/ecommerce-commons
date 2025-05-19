package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var errEntityNotFound = errors.New("entity not found in database")

type Store interface {

	// can return EntityNotFoundError
	FetchAndLock(ctx context.Context) (OutboxEntity, error)

	Create(ctx context.Context, payload string, key string, topic string) (OutboxEntity, error)

	UpdateAsSentByIds(ctx context.Context, ids []primitive.ObjectID) error
}

type store struct {
	collection collection
}

func NewStore(collection collection) Store {
	return &store{collection}
}

func (r *store) FetchAndLock(ctx context.Context) (OutboxEntity, error) {
	var message OutboxEntity

	opts := options.FindOneAndUpdate().SetSort(bson.M{"lockExpiresAt": 1, "createdAt": 1}).SetReturnDocument(options.After)

	filter := bson.M{
		"$or": []bson.M{
			{"lockExpiresAt": bson.M{"$lt": time.Now()}},
			{"lockExpiresAt": bson.M{"$eq": nil}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"lockExpiresAt":  time.Now().Add(30 * time.Second),
			"attemptsToSend": bson.M{"$inc": 1},
		},
	}

	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&message)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return OutboxEntity{}, fmt.Errorf("failed to fetch outbox entity: %w", errEntityNotFound)
		}
		return OutboxEntity{}, fmt.Errorf("failed to fetch outbox entity: %w", err)
	}

	return message, nil
}

func (r *store) Create(ctx context.Context, payload string, key string, topic string) (OutboxEntity, error) {
	entity := OutboxEntity{
		Payload:        payload,
		Key:            key,
		Topic:          topic,
		CreatedAt:      time.Now().UTC(),
		Status:         "PROCESSING",
		LockExpiresAt:  time.Now().Add(30 * time.Second),
		AttemptsToSend: 0,
	}
	result, err := r.collection.InsertOne(ctx, entity)
	if err != nil {
		return OutboxEntity{}, fmt.Errorf("failed to insert outbox entity: %w", err)
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return OutboxEntity{}, fmt.Errorf("failed to cast inserted ID %v to ObjectID: %w", result.InsertedID, err)
	}
	entity.ID = id
	return entity, nil
}

func (r *store) UpdateAsSentByIds(ctx context.Context, ids []primitive.ObjectID) error {
	_, err := r.collection.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"status": "SENT"}, "$unset": bson.M{"lockExpiresAt": ""}})
	if err != nil {
		return fmt.Errorf("failed to update outbox messages: %w", err)
	}
	return nil
}
