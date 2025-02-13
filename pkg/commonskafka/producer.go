package commonskafka

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log/slog"
)

type ProducerInterface interface {
	SendMessage(topic string, message []byte) error
	Close()
}

type KafkaProducer struct {
	producer *kafka.Producer
	conf     *KafkaConf
}

func NewKafkaProducer(conf *KafkaConf) (*KafkaProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": conf.Brokers})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					slog.Error(fmt.Sprintf("failed to publish message to: %v\n", ev.TopicPartition))
				} else {
					slog.Info(fmt.Sprintf("message published to: %v\n", ev.TopicPartition))
				}
			}
		}
	}()

	return &KafkaProducer{producer: p, conf: conf}, nil
}

func (kp *KafkaProducer) SendMessage(topic string, message []byte) error {
	err := kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}
	return nil
}

func (kp *KafkaProducer) Close() {
	kp.producer.Flush(15000)
	kp.producer.Close()
}
