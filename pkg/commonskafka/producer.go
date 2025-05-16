package commonskafka

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type ProducerInterface interface {
	Produce(message *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

type KafkaProducer struct {
	producer *kafka.Producer
	conf     *KafkaConf
	log      *zap.Logger
}

func NewKafkaProducer(conf *KafkaConf, log *zap.Logger) (*KafkaProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": conf.Brokers})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Error(fmt.Sprintf("failed to publish message to: %v", ev.TopicPartition))
				} else {
					log.Info(fmt.Sprintf("message published to: %v", ev.TopicPartition))
				}
			}
		}
	}()

	return &KafkaProducer{producer: p, conf: conf}, nil
}

func (kp *KafkaProducer) Produce(message *kafka.Message, deliveryChan chan kafka.Event) error {
	err := kp.producer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", message.TopicPartition, err)
	}
	return nil
}

func (kp *KafkaProducer) Close() {
	kp.producer.Flush(15000)
	kp.producer.Close()
}
