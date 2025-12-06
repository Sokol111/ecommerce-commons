package outbox

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type sender struct {
	producer        producer.Producer
	entitiesChan    <-chan *outboxEntity
	deliveryChan    chan kafka.Event
	logger          *zap.Logger
	tracePropagator tracePropagator
}

func newSender(
	producer producer.Producer,
	entitiesChan chan *outboxEntity,
	deliveryChan chan kafka.Event,
	logger *zap.Logger,
	tracePropagator tracePropagator,
) *sender {
	return &sender{
		producer:        producer,
		entitiesChan:    entitiesChan,
		deliveryChan:    deliveryChan,
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
			if err := s.send(entity); err != nil {
				s.logger.Error("failed to send outbox message",
					zap.String("id", entity.ID),
					zap.Error(err))
				continue
			}
			s.logger.Debug("outbox sent to kafka", zap.String("id", entity.ID))
		}
	}
}

func (s *sender) send(entity *outboxEntity) error {
	_, span, kafkaHeaders := s.tracePropagator.StartKafkaProducerSpan(entity.Headers, entity.Topic, entity.ID)
	defer span.End()

	err := s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &entity.Topic, Partition: kafka.PartitionAny},
		Opaque:         entity.ID,
		Value:          entity.Payload,
		Key:            []byte(entity.Key),
		Headers:        kafkaHeaders,
	}, s.deliveryChan)

	if err != nil {
		return fmt.Errorf("failed to send outbox message with id %v: %w", entity.ID, err)
	}

	return nil
}
