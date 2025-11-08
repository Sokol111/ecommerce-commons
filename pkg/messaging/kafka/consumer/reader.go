package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type reader struct {
	consumer     *kafka.Consumer
	topic        string
	messagesChan chan<- *kafka.Message
	log          *zap.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func newReader(
	consumer *kafka.Consumer,
	topic string,
	messagesChan chan<- *kafka.Message,
	log *zap.Logger,
) *reader {
	return &reader{
		consumer:     consumer,
		topic:        topic,
		messagesChan: messagesChan,
		log:          log,
	}
}

func (r *reader) start() {
	r.log.Info("starting reader")

	err := r.consumer.SubscribeTopics([]string{r.topic}, nil)
	if err != nil {
		r.log.Error("failed to subscribe to topic", zap.String("topic", r.topic), zap.Error(err))
		return
	}

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

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			msg, err := r.consumer.ReadMessage(5 * time.Second)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) && kafkaErr.IsTimeout() {
					continue
				}
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
