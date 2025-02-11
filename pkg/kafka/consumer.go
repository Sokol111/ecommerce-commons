package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log"
)

type ConsumerInterface interface {
	Consume(ctx context.Context) error
	Close()
}

type KafkaConsumer struct {
	consumer *kafka.Consumer
	topics   []string
}

func NewKafkaConsumer(conf *KafkaConf, groupID string, topics []string) (*KafkaConsumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": conf.Brokers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, fmt.Errorf("не вдалося створити консьюмера: %w", err)
	}

	err = c.SubscribeTopics(topics, nil)
	if err != nil {
		return nil, fmt.Errorf("не вдалося підписатися на теми: %w", err)
	}

	return &KafkaConsumer{
		consumer: c,
		topics:   topics,
	}, nil
}

func (kc *KafkaConsumer) Consume(ctx context.Context) error {
	log.Println("Kafka Consumer запущено...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka Consumer отримав сигнал завершення")
			return nil
		default:
			msg, err := kc.consumer.ReadMessage(-1) // Блокуючий виклик
			if err == nil {
				log.Printf("Отримано повідомлен ня з %s: %s\n", *msg.TopicPartition.Topic, string(msg.Value))
			} else if !err.(kafka.Error).IsTimeout() {
				fmt.Printf("Consumer error: %v (%v)\n", err, msg)
			}
		}
	}
}

// Close закриває консьюмера
func (kc *KafkaConsumer) Close() {
	kc.consumer.Close()
}
