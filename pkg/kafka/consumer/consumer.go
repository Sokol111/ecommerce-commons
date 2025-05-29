package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type Consumer interface {
	Start() error
	Stop(ctx context.Context) error
}

type consumer[T any] struct {
	consumer *kafka.Consumer
	topic    string
	handler  Handler[T]
	log      *zap.Logger

	messagesChan chan *kafka.Message

	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc

	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

func NewConsumer[T any](brokers, groupID, topic string, autoOffsetReset string, handler Handler[T], log *zap.Logger) (Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"group.id":                 groupID,
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
		"auto.commit.interval.ms":  3000,
		"auto.offset.reset":        autoOffsetReset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer, topic: %s: %w", topic, err)
	}

	return &consumer[T]{
		consumer:     c,
		topic:        topic,
		handler:      handler,
		log:          log.With(zap.String("component", "consumer"), zap.String("topic", topic)),
		messagesChan: make(chan *kafka.Message, 100),
	}, nil
}

func (c *consumer[T]) Start() error {
	var startErr error
	c.startOnce.Do(func() {
		c.log.Info("starting consumer workers")
		c.ctx, c.cancelFunc = context.WithCancel(context.Background())

		err := c.consumer.SubscribeTopics([]string{c.topic}, nil)
		if err != nil {
			startErr = fmt.Errorf("failed to subscribe to topic %s: %w", c.topic, err)
			return
		}
		c.wg.Add(2)
		go c.startReadingWorker()
		go c.startProcessingWorker()
		c.started.Store(true)
		c.log.Info("consumer started")
	})
	return startErr
}

func (c *consumer[T]) startReadingWorker() {
	defer func() {
		defer c.log.Info("fetching worker stopped")
		c.wg.Done()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.consumer.ReadMessage(5 * time.Second)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) && kafkaErr.IsTimeout() {
					continue
				}
				c.log.Error("failed to read message", zap.Error(err))
				continue
			}

			select {
			case <-c.ctx.Done():
				return
			case c.messagesChan <- msg:
			}
		}
	}
}

func (c *consumer[T]) startProcessingWorker() {
	defer func() {
		c.log.Info("processing worker stopped")
		c.wg.Done()
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.messagesChan:
			if c.ctx.Err() != nil {
				return
			}
			c.handleMessage(msg)
		}
	}
}

func (c *consumer[T]) handleMessage(message *kafka.Message) {
	for attempt := 1; c.ctx.Err() == nil; attempt++ {

		var event event.Event[T]
		err := c.parseMessage(message, &event)
		if err != nil {
			c.log.Error("failed to parse message",
				zap.String("key", string(message.Key)),
				zap.String("value", string(message.Value)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.Error(err))
			return
		}

		err = c.validateEvent(&event)
		if err != nil {
			c.log.Error("failed to validate event",
				zap.String("key", string(message.Key)),
				zap.String("value", string(message.Value)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.String("event", fmt.Sprintf("%v", event)),
				zap.Error(err))
			return
		}

		err = c.handler.Validate(event.Payload)
		if err != nil {
			c.log.Error("failed to validate event payload",
				zap.String("key", string(message.Key)),
				zap.String("value", string(message.Value)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int32("offset", int32(message.TopicPartition.Offset)),
				zap.String("event", fmt.Sprintf("%v", event)),
				zap.Error(err))
			return
		}

		err = c.handler.Process(c.ctx, &event)

		if err != nil {
			c.log.Error("failed to process message", zap.Error(err))
			sleep(c.ctx, backoffDuration(attempt, 10*time.Second))
			continue
		}

		for ; c.ctx.Err() == nil; attempt++ {
			_, err = c.consumer.StoreMessage(message)

			if err != nil {
				c.log.Error("failed to store message", zap.Error(err))
				sleep(c.ctx, backoffDuration(attempt, 10*time.Second))
			} else {
				return
			}
		}
	}
}

func (c *consumer[T]) parseMessage(message *kafka.Message, e *event.Event[T]) error {
	if message == nil || len(message.Value) == 0 {
		return fmt.Errorf("empty Kafka message")
	}

	err := json.Unmarshal(message.Value, e)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return nil
}

func (c *consumer[T]) validateEvent(e *event.Event[T]) error {
	if e == nil {
		return fmt.Errorf("empty event")
	}

	if e.EventID == "" {
		return fmt.Errorf("empty EventID")
	}

	if e.Payload == nil {
		return fmt.Errorf("empty Payload")
	}

	if e.CreatedAt.IsZero() {
		return fmt.Errorf("empty CreatedAt")
	}

	if e.Type == "" {
		return fmt.Errorf("empty Type")
	}

	if e.Source == "" {
		return fmt.Errorf("empty Source")
	}

	if e.Type == "" {
		return fmt.Errorf("empty Type")
	}

	if e.Version == 0 {
		return fmt.Errorf("empty Version")
	}

	// if e.TraceId == "" {
	// 	return fmt.Errorf("empty TraceId")
	// }

	return nil
}

func (c *consumer[T]) Stop(ctx context.Context) error {
	if !c.started.Load() {
		c.log.Warn("consumer not started, skipping stop")
		return nil
	}

	var resultErr error

	c.stopOnce.Do(func() {
		c.log.Info("stopping consumer")
		c.cancelFunc()

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.wg.Wait()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			c.log.Warn("shutdown timed out waiting for workers", zap.Error(ctx.Err()))
		}

		if _, commitErr := c.consumer.Commit(); commitErr != nil {
			if commitErr.(kafka.Error).Code() != kafka.ErrNoOffset {
				c.log.Warn("failed to commit offsets", zap.Error(commitErr))
			}
		}

		if closeErr := c.consumer.Close(); closeErr != nil {
			c.log.Error("failed to close consumer", zap.Error(closeErr))
			resultErr = closeErr
		}

		c.log.Info("consumer stopped")
	})

	return resultErr
}

func backoffDuration(attempt int, max time.Duration) time.Duration {
	var backoff time.Duration
	var duration = time.Duration(math.Pow(2, float64(attempt))) * time.Second
	if duration > max {
		backoff = max
	}
	return backoff
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}
