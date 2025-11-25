package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type reader struct {
	consumer     *kafka.Consumer
	topic        string
	messagesChan chan<- *kafka.Message
	log          *zap.Logger
	readiness    health.ReadinessWaiter

	ctx          context.Context
	cancelFunc   context.CancelFunc
	wg           sync.WaitGroup
	errorTracker *errorTracker
}

func newReader(
	consumer *kafka.Consumer,
	topic string,
	messagesChan chan<- *kafka.Message,
	log *zap.Logger,
	readiness health.ReadinessWaiter,
) *reader {
	return &reader{
		consumer:     consumer,
		topic:        topic,
		messagesChan: messagesChan,
		log:          log,
		readiness:    readiness,
		errorTracker: newErrorTracker(log),
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
}

func (r *reader) run() {
	defer func() {
		r.log.Info("reader worker stopped")
		r.wg.Done()
	}()

	// Wait for traffic readiness before starting to read messages
	r.log.Info("waiting for traffic readiness before reading messages")
	if err := r.readiness.WaitForTrafficReady(r.ctx); err != nil {
		r.log.Error("context cancelled while waiting for traffic readiness")
		return
	}
	r.log.Info("traffic readiness achieved, starting to read messages")

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			msg, err := r.consumer.ReadMessage(30 * time.Second)

			// Classify the error
			readerErr := wrapReaderError(err)

			// Success - no error
			if readerErr == nil {
				select {
				case <-r.ctx.Done():
					return
				case r.messagesChan <- msg:
				}
				continue
			}

			switch {
			case readerErr.isFatal():
				// Fatal error - stop consumer
				r.log.Error(readerErr.description, zap.Error(readerErr))
				r.cancelFunc()
				return

			case readerErr.isTimeout():
				// Timeout - silent retry
				continue

			case readerErr.isTemporary():
				// Temporary error - log and retry
				r.errorTracker.logReaderError(readerErr)
				continue

			default:
				// Non-Kafka or unknown error - log and continue
				r.log.Error("failed to read message", zap.Error(readerErr))
				continue
			}
		}
	}
}
