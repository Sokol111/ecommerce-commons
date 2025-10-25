package producer

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type Producer interface {
	Produce(message *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

type producer struct {
	producer *kafka.Producer
	log      *zap.Logger
}

func newProducer(conf config.Config, log *zap.Logger) (Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": conf.Brokers})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// go func() {
	// 	for e := range p.Events() {
	// 		switch ev := e.(type) {
	// 		case *kafka.Message:
	// 			if ev.TopicPartition.Error != nil {
	// 				log.Error(fmt.Sprintf("failed to publish message to: %v", ev.TopicPartition))
	// 			} else {
	// 				log.Info(fmt.Sprintf("message published to: %v", ev.TopicPartition))
	// 			}
	// 		}
	// 	}
	// }()

	return &producer{producer: p, log: log}, nil
}

func (p *producer) Produce(message *kafka.Message, deliveryChan chan kafka.Event) error {
	err := p.producer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", message.TopicPartition, err)
	}
	return nil
}

func (p *producer) Close() {
	p.producer.Close()
}
