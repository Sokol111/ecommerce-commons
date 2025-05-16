package commonskafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
	"math"
	"time"
)

type ConsumerInterface interface {
	Consume(ctx context.Context) error
	Close()
}

type KafkaConsumer struct {
	consumer *kafka.Consumer
	topic    string
	handler  MessageHandler
	log      *zap.Logger
}

func NewKafkaConsumer(brokers, groupID, topic string, handler MessageHandler, log *zap.Logger) (*KafkaConsumer, error) {
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

	return &KafkaConsumer{
		consumer: c,
		topic:    topic,
		handler:  handler,
		log:      log,
	}, nil
}

func (kc *KafkaConsumer) StartConsuming(ctx context.Context) {
	go func() {
		kc.log.Info(fmt.Sprintf("kafka consumer for topic %s is started", kc.topic))

		for {
			select {
			case <-ctx.Done():
				kc.log.Info(fmt.Sprintf("kafka consumer for topic %s stopped", kc.topic))
				return
			default:
				msg, err := kc.consumer.ReadMessage(5 * time.Second)
				if err != nil {
					var kafkaErr kafka.Error
					if errors.As(err, &kafkaErr) && kafkaErr.IsTimeout() {
						continue
					}
					kc.log.Error(fmt.Sprintf("failed to read message from topic %s: %v", kc.topic, err))
					continue
				}

				for attempt := 1; ; attempt++ {
					if ctx.Err() != nil {
						kc.log.Info(fmt.Sprintf("kafka consumer for topic %s stopped", kc.topic))
						return
					}

					err := kc.handler.HandleMessage(ctx, msg)
					if err == nil {
						_, commitErr := kc.consumer.CommitMessage(msg)
						if commitErr == nil {
							break
						}
						kc.log.Error(fmt.Sprintf("failed to commit message for topic %s", kc.topic), zap.Error(commitErr))
					}

					time.Sleep(time.Duration(math.Min(float64(attempt*2), 10)) * time.Second)
				}
			}
		}
	}()
}

func (kc *KafkaConsumer) Close() {
	kc.consumer.Close()
}
