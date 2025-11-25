package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type errorTracker struct {
	count     int
	firstSeen time.Time
	lastWarn  time.Time
}

type reader struct {
	consumer     *kafka.Consumer
	topic        string
	messagesChan chan<- *kafka.Message
	log          *zap.Logger
	readiness    health.ReadinessWaiter

	ctx           context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
	errorCounters map[string]*errorTracker
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

func (r *reader) logRetriableError(errorType string, msg string, fields ...zap.Field) {
	if r.errorCounters == nil {
		r.errorCounters = make(map[string]*errorTracker)
	}

	tracker, exists := r.errorCounters[errorType]
	if !exists {
		tracker = &errorTracker{firstSeen: time.Now()}
		r.errorCounters[errorType] = tracker
	}
	tracker.count++

	shouldWarn := tracker.lastWarn.IsZero() || time.Since(tracker.lastWarn) > 5*time.Minute

	if shouldWarn {
		r.log.Warn(msg, append(fields,
			zap.Int("attempts", tracker.count),
			zap.Duration("duration", time.Since(tracker.firstSeen)))...)
		tracker.lastWarn = time.Now()
	} else {
		r.log.Debug(msg, fields...)
	}
}

func (r *reader) run() {
	defer func() {
		r.log.Info("reader worker stopped")
		r.wg.Done()
	}()

	// Wait for traffic readiness before starting to read messages
	r.log.Info("waiting for traffic readiness before reading messages")
	if err := r.readiness.WaitForTrafficReady(r.ctx); err != nil {
		r.log.Info("context cancelled while waiting for traffic readiness")
		return
	}
	r.log.Info("traffic readiness achieved, starting to read messages")

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			msg, err := r.consumer.ReadMessage(30 * time.Second)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) {
					switch {
					case kafkaErr.IsTimeout():
						continue

					case kafkaErr.IsFatal():
						r.log.Error("fatal kafka error - consumer instance is no longer operable",
							zap.Error(err),
							zap.String("error_code", kafkaErr.Code().String()))
						r.cancelFunc()
						return

					case kafkaErr.Code() == kafka.ErrUnknownTopicOrPart:
						// Topic doesn't exist yet - wait longer before retrying
						r.logRetriableError("topic_not_found", "topic not available, waiting for topic creation")
						continue

					case kafkaErr.Code() == kafka.ErrTransport ||
						kafkaErr.Code() == kafka.ErrAllBrokersDown ||
						kafkaErr.Code() == kafka.ErrNetworkException:
						// Connection issues - these are usually temporary
						r.logRetriableError("broker_connection", "broker connection issue, retrying", zap.Error(err))
						continue

					case kafkaErr.Code() == kafka.ErrLeaderNotAvailable ||
						kafkaErr.Code() == kafka.ErrNotLeaderForPartition:
						// Leader election or rebalance in progress
						r.logRetriableError("leader_election", "partition leader changing, retrying")
						continue

					case kafkaErr.IsRetriable():
						// Всі інші retriable помилки
						r.logRetriableError("retriable_error",
							"retriable kafka error, retrying",
							zap.String("topic", r.topic),
							zap.String("error_code", kafkaErr.Code().String()),
							zap.Error(err))
						continue
					}
				}

				// All other errors - log as error
				r.log.Error("failed to read message", zap.Error(err))
				continue
			}

			// Успішно прочитали повідомлення - скидаємо всі лічильники помилок
			r.errorCounters = nil

			select {
			case <-r.ctx.Done():
				return
			case r.messagesChan <- msg:
			}
		}
	}
}
