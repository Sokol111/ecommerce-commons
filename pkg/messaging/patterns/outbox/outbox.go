package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

type OutboxMessage struct {
	Message any               // The actual message payload to be sent
	EventID string            // Unique event identifier - used as outbox record ID for idempotency (prevents duplicate messages)
	Key     string            // Kafka partition key for ordering guarantees
	Headers map[string]string // Kafka headers for trace propagation, event_type, etc.
}

type Outbox interface {
	Create(ctx context.Context, msg OutboxMessage) (SendFunc, error)
}

type outbox struct {
	store        Store
	logger       *zap.Logger
	entitiesChan chan<- *outboxEntity
	typeMapping  *mapping.TypeMapping
	serializer   serialization.Serializer
}

func newOutbox(logger *zap.Logger, store Store, entitiesChan chan<- *outboxEntity, typeMapping *mapping.TypeMapping, serializer serialization.Serializer) Outbox {
	return &outbox{
		store:        store,
		logger:       logger,
		entitiesChan: entitiesChan,
		typeMapping:  typeMapping,
		serializer:   serializer,
	}
}

type SendFunc func(ctx context.Context) error

func (o *outbox) Create(ctx context.Context, msg OutboxMessage) (SendFunc, error) {
	// Extract trace context from ctx and inject into headers
	msg.Headers = o.injectTraceContext(ctx, msg.Headers)

	mapping, err := o.typeMapping.GetByValue(msg.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to get type mapping for outbox message: %w", err)
	}

	serializedMsg, err := o.serializer.Serialize(msg.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize outbox message: %w", err)
	}

	entity, err := o.store.Create(ctx, serializedMsg, msg.EventID, msg.Key, mapping.Topic, msg.Headers)
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

func (o *outbox) injectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}

	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier(headers)
	propagator.Inject(ctx, carrier)

	return headers
}

func (o *outbox) log(ctx context.Context) *zap.Logger {
	return logger.Get(ctx).With(zap.String("component", "outbox"))
}
