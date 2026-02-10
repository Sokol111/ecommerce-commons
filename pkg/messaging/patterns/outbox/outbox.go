package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	"go.uber.org/zap"
)

// Message represents a message to be sent via the outbox pattern.
type Message struct {
	Event   events.Event      // Event payload - must implement events.Event interface
	Key     string            // Kafka partition key for ordering guarantees
	Headers map[string]string // Kafka headers for trace propagation, etc.
}

// Outbox defines the interface for creating outbox messages.
type Outbox interface {
	Create(ctx context.Context, msg Message) (SendFunc, error)
}

type outbox struct {
	outboxRepository  repository
	logger            *zap.Logger
	entitiesChan      chan<- *outboxEntity
	serializer        serde.Serializer
	tracePropagator   tracePropagator
	metadataPopulator events.MetadataPopulator
}

func newOutbox(logger *zap.Logger, outboxRepository repository, entitiesChan chan *outboxEntity, serializer serde.Serializer, tracePropagator tracePropagator, metadataPopulator events.MetadataPopulator) Outbox {
	return &outbox{
		outboxRepository:  outboxRepository,
		logger:            logger,
		entitiesChan:      entitiesChan,
		serializer:        serializer,
		tracePropagator:   tracePropagator,
		metadataPopulator: metadataPopulator,
	}
}

// SendFunc is a function that triggers the actual message delivery after transaction commit.
type SendFunc func(ctx context.Context) error

func (o *outbox) Create(ctx context.Context, msg Message) (SendFunc, error) {
	// Populate event metadata automatically (EventID, EventType, Source, Timestamp, TraceID)
	eventID := o.metadataPopulator.PopulateMetadata(ctx, msg.Event)

	// Save trace context into headers for storage in outbox
	msg.Headers = o.tracePropagator.SaveTraceContext(ctx, msg.Headers)

	// Serialize the event - topic is obtained from event.GetTopic()
	serializedMsg, err := o.serializer.Serialize(msg.Event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize outbox message: %w", err)
	}

	// Get topic from the event itself (self-describing)
	topic := msg.Event.GetTopic()

	entity, err := o.outboxRepository.Create(ctx, serializedMsg, eventID, msg.Key, topic, msg.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox message: %w", err)
	}

	o.log(ctx).Debug("outbox created", zap.String("id", entity.ID))

	return o.createSendFunc(entity), nil
}

func (o *outbox) createSendFunc(entity *outboxEntity) SendFunc {
	return func(ctx context.Context) error {
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
	}
}

func (o *outbox) log(ctx context.Context) *zap.Logger {
	return logger.Get(ctx).With(zap.String("component", "outbox"))
}
