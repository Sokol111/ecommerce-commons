package messaging

import (
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/patterns/outbox"
	"go.uber.org/fx"
)

// messagingOptions holds internal configuration for the messaging module.
type messagingOptions struct {
	kafkaConfig *config.Config
}

// MessagingOption is a functional option for configuring the messaging module.
type MessagingOption func(*messagingOptions)

// WithKafkaConfig provides a static Kafka Config (useful for tests).
// When set, the Kafka configuration will not be loaded from viper.
func WithKafkaConfig(cfg config.Config) MessagingOption {
	return func(opts *messagingOptions) {
		opts.kafkaConfig = &cfg
	}
}

// NewMessagingModule provides messaging functionality: kafka, outbox, consumer, producer.
//
// Options:
//   - WithKafkaConfig: provide static Kafka Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from viper
//	messaging.NewMessagingModule()
//
//	// Testing - with static config
//	messaging.NewMessagingModule(
//	    messaging.WithKafkaConfig(config.Config{...}),
//	)
func NewMessagingModule(opts ...MessagingOption) fx.Option {
	cfg := &messagingOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		kafkaConfigModule(cfg),
		producer.NewProducerModule(),
		avro.NewAvroModule(),
		outbox.NewOutboxModule(),
	)
}

func kafkaConfigModule(cfg *messagingOptions) fx.Option {
	if cfg.kafkaConfig != nil {
		return config.NewKafkaConfigModule(config.WithKafkaConfig(*cfg.kafkaConfig))
	}
	return config.NewKafkaConfigModule()
}
