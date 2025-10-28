package outbox

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type sender struct {
	producer     producer.Producer
	entitiesChan <-chan *outboxEntity
	deliveryChan chan kafka.Event
	logger       *zap.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func newSender(
	producer producer.Producer,
	entitiesChan <-chan *outboxEntity,
	deliveryChan chan kafka.Event,
	logger *zap.Logger,
) *sender {
	return &sender{
		producer:     producer,
		entitiesChan: entitiesChan,
		deliveryChan: deliveryChan,
		logger:       logger.With(zap.String("component", "outbox")),
	}
}

func provideSender(lc fx.Lifecycle, producer producer.Producer, channels *channels, logger *zap.Logger) *sender {
	s := newSender(producer, channels.entities, channels.delivery, logger)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			s.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.stop()
			return nil
		},
	})

	return s
}

func (s *sender) start() {
	s.logger.Info("starting sender")
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	go s.run()
}

func (s *sender) stop() {
	s.logger.Info("stopping sender")
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

func (s *sender) run() {
	defer s.logger.Info("sender worker stopped")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		select {
		case <-s.ctx.Done():
			return
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
	err := s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &entity.Topic, Partition: kafka.PartitionAny},
		Opaque:         entity.ID,
		Value:          []byte(entity.Payload),
		Key:            []byte(entity.Key),
	}, s.deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send outbox message with id %v: %w", entity.ID, err)
	}
	return nil
}
