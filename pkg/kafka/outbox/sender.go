package outbox

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type Sender struct {
	producer        producer.Producer
	entitiesChan    <-chan *OutboxEntity
	confirmChan     chan<- ConfirmResult
	logger          *zap.Logger
	tracePropagator TracePropagator
}

func NewSender(
	producer producer.Producer,
	entitiesChan chan *OutboxEntity,
	confirmChan chan ConfirmResult,
	logger *zap.Logger,
	tracePropagator TracePropagator,
) *Sender {
	return &Sender{
		producer:        producer,
		entitiesChan:    entitiesChan,
		confirmChan:     confirmChan,
		logger:          logger,
		tracePropagator: tracePropagator,
	}
}

func (s *Sender) Run(ctx context.Context) error {
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

func (s *Sender) send(ctx context.Context, entity *OutboxEntity) {
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
		confirmChan <- ConfirmResult{ID: entityID, Err: err}
	})
}
