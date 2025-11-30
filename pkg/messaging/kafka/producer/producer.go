package producer

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type Producer interface {
	Produce(message *kafka.Message, deliveryChan chan kafka.Event) error
}

type producer struct {
	producer *kafka.Producer
	log      *zap.Logger
}

func newProducer(kafkaProducer *kafka.Producer, log *zap.Logger) Producer {
	return &producer{producer: kafkaProducer, log: log}
}

func (p *producer) Produce(message *kafka.Message, deliveryChan chan kafka.Event) error {
	err := p.producer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", message.TopicPartition, err)
	}
	return nil
}
