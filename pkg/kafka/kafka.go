package kafka

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaConf struct {
	Brokers string `mapstructure:"brokers"`
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *kafka.Message) error
}
