package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Config struct {
	Brokers       string `mapstructure:"brokers"`
	ConsumerGroup string `mapstructure:"consumer-group"`
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *kafka.Message) error
}
