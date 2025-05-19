package kafka

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type Consumer interface {
	StartConsuming(ctx context.Context)
	Close()
}

type consumer struct {
	consumer *kafka.Consumer
	topic    string
	handler  MessageHandler
	log      *zap.Logger
}

func NewConsumer(brokers, groupID, topic string, handler MessageHandler, log *zap.Logger) (Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	return &consumer{
		consumer: c,
		topic:    topic,
		handler:  handler,
		log:      log,
	}, nil
}

func (c *consumer) StartConsuming(ctx context.Context) {
	go func() {
		c.log.Info(fmt.Sprintf("kafka consumer for topic %s is started", c.topic))

		for {
			select {
			case <-ctx.Done():
				c.log.Info(fmt.Sprintf("kafka consumer for topic %s stopped", c.topic))
				return
			default:
				msg, err := c.consumer.ReadMessage(5 * time.Second)
				if err != nil {
					var kafkaErr kafka.Error
					if errors.As(err, &kafkaErr) && kafkaErr.IsTimeout() {
						continue
					}
					c.log.Error(fmt.Sprintf("failed to read message from topic %s: %v", c.topic, err))
					continue
				}

				for attempt := 1; ; attempt++ {
					if ctx.Err() != nil {
						c.log.Info(fmt.Sprintf("kafka consumer for topic %s stopped", c.topic))
						return
					}

					err := c.handler.HandleMessage(ctx, msg)
					if err == nil {
						_, commitErr := c.consumer.CommitMessage(msg)
						if commitErr == nil {
							break
						}
						c.log.Error(fmt.Sprintf("failed to commit message for topic %s", c.topic), zap.Error(commitErr))
					}

					time.Sleep(time.Duration(math.Min(float64(attempt*2), 10)) * time.Second)
				}
			}
		}
	}()
}

func (kc *consumer) Close() {
	kc.consumer.Close()
}
