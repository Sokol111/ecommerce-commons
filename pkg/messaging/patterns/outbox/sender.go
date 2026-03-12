package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type sender struct {
	producer        producer.Producer
	entitiesChan    <-chan *outboxEntity
	confirmChan     chan<- confirmResult
	logger          *zap.Logger
	tracePropagator tracePropagator
}

func newSender(
	producer producer.Producer,
	entitiesChan chan *outboxEntity,
	confirmChan chan confirmResult,
	logger *zap.Logger,
	tracePropagator tracePropagator,
) *sender {
	return &sender{
		producer:        producer,
		entitiesChan:    entitiesChan,
		confirmChan:     confirmChan,
		logger:          logger,
		tracePropagator: tracePropagator,
	}
}

func (s *sender) Run(ctx context.Context) error {
	defer s.logger.Info("sender worker stopped")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		select {
		case <-ctx.Done():
			return nil
		case entity := <-s.entitiesChan:
			s.send(ctx, entity)
			s.logger.Debug("outbox sent to kafka", zap.String("id", entity.ID))
		}
	}
}

func (s *sender) send(ctx context.Context, entity *outboxEntity) {
	_, span, kafkaHeaders := s.tracePropagator.StartKafkaProducerSpan(entity.Headers, entity.Topic, entity.ID)
	defer span.End()

	record := &kgo.Record{
		Topic:   entity.Topic,
		Key:     []byte(entity.Key),
		Value:   entity.Payload,
		Headers: kafkaHeaders,
	}

	entityID := entity.ID
	confirmChan := s.confirmChan
	s.producer.Produce(ctx, record, func(_ *kgo.Record, err error) {
		confirmChan <- confirmResult{ID: entityID, Err: err}
	})
}
