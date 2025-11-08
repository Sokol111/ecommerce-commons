package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/zap"
)

type OutboxMessage struct {
	Payload []byte // Pre-serialized event bytes (e.g., Avro, Protobuf, JSON) - ready to send to Kafka
	EventID string // Unique event identifier - used as outbox record ID for idempotency (prevents duplicate messages)
	Key     string // Kafka partition key for ordering guarantees
	Topic   string // Kafka topic name
}

type Outbox interface {
	Create(ctx context.Context, msg OutboxMessage) (SendFunc, error)
}

type outbox struct {
	store        Store
	logger       *zap.Logger
	entitiesChan chan<- *outboxEntity
}

func newOutbox(logger *zap.Logger, store Store, entitiesChan chan<- *outboxEntity) Outbox {
	return &outbox{
		store:        store,
		logger:       logger.With(zap.String("component", "outbox")),
		entitiesChan: entitiesChan,
	}
}

type SendFunc func(ctx context.Context) error

func (o *outbox) Create(ctx context.Context, msg OutboxMessage) (SendFunc, error) {
	entity, err := o.store.Create(ctx, msg.Payload, msg.EventID, msg.Key, msg.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox message: %w", err)
	}

	o.log(ctx).Debug("outbox created", zap.String("id", entity.ID))

	return SendFunc(func(ctx context.Context) error {
		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return fmt.Errorf("outbox didn't sent: %w", ctx.Err())
		case o.entitiesChan <- entity:
			return nil
		case <-timer.C:
			o.log(ctx).Warn("entitiesChan is full, dropping message", zap.String("id", entity.ID))
			return fmt.Errorf("entitiesChan is full")
		}
	}), nil
}

func (o *outbox) log(ctx context.Context) *zap.Logger {
	return logger.FromContext(ctx).With(zap.String("component", "outbox"))
}
