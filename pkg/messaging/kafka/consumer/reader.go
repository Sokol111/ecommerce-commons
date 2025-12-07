package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
)

// messageReader is an interface for reading messages from Kafka.
type messageReader interface {
	ReadMessage(timeout time.Duration) (*kafka.Message, error)
}

type reader struct {
	consumer     messageReader
	messagesChan chan<- *kafka.Message
	log          *zap.Logger
	throttler    *logger.LogThrottler
}

func newReader(
	consumer messageReader,
	messagesChan chan *kafka.Message,
	log *zap.Logger,
) *reader {
	return &reader{
		consumer:     consumer,
		messagesChan: messagesChan,
		log:          log,
		throttler:    logger.NewLogThrottler(log, 0),
	}
}

func (r *reader) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := r.consumer.ReadMessage(30 * time.Second)
			if err == nil {
				select {
				case <-ctx.Done():
					return nil
				case r.messagesChan <- msg:
				}
				continue
			}

			var kafkaErr kafka.Error
			if !errors.As(err, &kafkaErr) {
				r.log.Error("non-kafka error reading message", zap.Error(err))
				continue
			}

			if kafkaErr.IsTimeout() {
				continue
			}

			if kafkaErr.IsFatal() {
				r.log.Error("fatal kafka error - consumer instance is no longer operable", zap.Error(err))
				return err // trigger application shutdown
			}

			if kafkaErr.Code() == kafka.ErrUnknownTopicOrPart {
				r.throttler.Warn("topic_not_available", "topic not available, waiting for topic creation", zap.Error(err))
				continue
			}

			if kafkaErr.IsRetriable() {
				r.throttler.Warn("retriable", "retriable kafka error, retrying", zap.Error(err))
				continue
			}

			r.log.Error("unknown kafka error", zap.Error(err))
		}
	}
}
