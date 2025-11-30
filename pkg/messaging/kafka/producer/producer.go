package producer

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// Producer is the interface for sending messages to Kafka.
type Producer interface {
	Produce(message *kafka.Message, deliveryChan chan kafka.Event) error
}

// kafkaProducer is the interface that wraps the Produce method of kafka.Producer.
type kafkaProducer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
}

type producer struct {
	kafkaProducer kafkaProducer
	log           *zap.Logger
}

func newProducer(kafkaProducer kafkaProducer, log *zap.Logger) Producer {
	return &producer{kafkaProducer: kafkaProducer, log: log}
}

func (p *producer) Produce(message *kafka.Message, deliveryChan chan kafka.Event) error {
	err := p.kafkaProducer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", message.TopicPartition, err)
	}
	return nil
}
