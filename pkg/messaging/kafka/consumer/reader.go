package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type reader struct {
	consumer     *kafka.Consumer
	topic        string
	messagesChan chan<- *kafka.Message
	log          *zap.Logger
	readiness    health.Readiness

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func newReader(
	consumer *kafka.Consumer,
	topic string,
	messagesChan chan<- *kafka.Message,
	log *zap.Logger,
	readiness health.Readiness,
) *reader {
	return &reader{
		consumer:     consumer,
		topic:        topic,
		messagesChan: messagesChan,
		log:          log,
		readiness:    readiness,
	}
}

func (r *reader) start() {
	r.log.Info("starting reader")

	r.ctx, r.cancelFunc = context.WithCancel(context.Background())
	r.wg.Add(1)
	go r.run()
}

func (r *reader) stop() {
	r.log.Info("stopping reader")
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	r.wg.Wait()

	if closeErr := r.consumer.Close(); closeErr != nil {
		r.log.Error("failed to close kafka consumer", zap.Error(closeErr))
	}

	r.log.Info("reader stopped")
}

func (r *reader) run() {
	defer func() {
		r.log.Info("reader worker stopped")
		r.wg.Done()
	}()

	// Wait for readiness before starting to read messages
	r.log.Info("waiting for readiness before reading messages")
	for !r.readiness.IsReady() {
		select {
		case <-r.ctx.Done():
			return
		default:
			sleep(r.ctx, 100*time.Millisecond)
		}
	}
	r.log.Info("readiness achieved, starting to read messages")

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			msg, err := r.consumer.ReadMessage(5 * time.Second)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) {
					// Normal timeout - continue without logging
					if kafkaErr.IsTimeout() {
						continue
					}

					// Topic doesn't exist yet - wait longer before retrying
					if kafkaErr.Code() == kafka.ErrUnknownTopicOrPart {
						r.log.Warn("topic not available, waiting for topic creation",
							zap.String("topic", r.topic))
						sleep(r.ctx, 10*time.Second)
						continue
					}

					// Connection issues - these are usually temporary
					if kafkaErr.Code() == kafka.ErrTransport ||
						kafkaErr.Code() == kafka.ErrAllBrokersDown ||
						kafkaErr.Code() == kafka.ErrNetworkException {
						r.log.Warn("broker connection issue, retrying",
							zap.String("topic", r.topic),
							zap.Error(err))
						sleep(r.ctx, 5*time.Second)
						continue
					}

					// Leader election or rebalance in progress
					if kafkaErr.Code() == kafka.ErrLeaderNotAvailable ||
						kafkaErr.Code() == kafka.ErrNotLeaderForPartition {
						r.log.Debug("partition leader changing, retrying",
							zap.String("topic", r.topic))
						sleep(r.ctx, 2*time.Second)
						continue
					}
				}

				// All other errors - log as error
				r.log.Error("failed to read message", zap.Error(err))
				continue
			}

			select {
			case <-r.ctx.Done():
				return
			case r.messagesChan <- msg:
			}
		}
	}
}
