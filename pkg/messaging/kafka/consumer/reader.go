package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type reader struct {
	consumer      *kafka.Consumer
	topic         string
	messagesChan  chan<- *kafka.Message
	log           *zap.Logger
	readiness     health.ReadinessWaiter
	errorLimiters sync.Map // map[string]*rate.Limiter
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
	}
}

func (r *reader) run(ctx context.Context) {
	// Wait for traffic readiness before starting to read messages
	r.log.Info("waiting for traffic readiness before reading messages")
	if err := r.readiness.WaitForTrafficReady(ctx); err != nil {
		r.log.Error("context cancelled while waiting for traffic readiness")
		return
	}
	r.log.Info("traffic readiness achieved, starting to read messages")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := r.consumer.ReadMessage(30 * time.Second)

			// Classify the error
			readerErr := wrapReaderError(err)

			// Success - no error
			if readerErr == nil {
				select {
				case <-ctx.Done():
					return
				case r.messagesChan <- msg:
				}
				continue
			}

			switch {
			case readerErr.isFatal():
				// Fatal error - stop consumer
				r.log.Error(readerErr.description, zap.Error(readerErr))
				return

			case readerErr.isTimeout():
				// Timeout - silent retry
				continue

			case readerErr.isTemporary():
				// Temporary error - log and retry
				r.logThrottled(readerErr.errorKey, readerErr.description, readerErr)
				continue

			default:
				// Non-Kafka or unknown error - log and continue
				r.log.Error("failed to read message", zap.Error(readerErr))
				continue
			}
		}
	}
}

// logThrottled logs as WARN once per 5 minutes per key, DEBUG otherwise
func (r *reader) logThrottled(key string, msg string, err error) {
	limiter := r.getErrorLimiter(key)

	if limiter.Allow() {
		r.log.Warn(msg, zap.Error(err))
	} else {
		r.log.Debug(msg, zap.Error(err))
	}
}

func (r *reader) getErrorLimiter(key string) *rate.Limiter {
	if limiter, ok := r.errorLimiters.Load(key); ok {
		return limiter.(*rate.Limiter)
	}

	// 1 event per 5 minutes, no burst
	limiter := rate.NewLimiter(rate.Every(5*60), 1)
	actual, _ := r.errorLimiters.LoadOrStore(key, limiter)
	return actual.(*rate.Limiter)
}
