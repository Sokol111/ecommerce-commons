package fxconfig

import (
	fx_kafka_config "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config/fxconfig"
	fx_kafkaproto "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/kafkaproto/fxconfig"
	fx_producer "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer/fxconfig"
	fx_outbox "github.com/Sokol111/ecommerce-commons/pkg/messaging/patterns/outbox/fxconfig"
	"go.uber.org/fx"
)

// messagingOptions holds internal configuration for the messaging module.
type messagingOptions struct {
	kafkaOpts []fx_kafka_config.Option
}

// Option is a functional option for configuring the messaging module.
type Option func(*messagingOptions)

// WithKafkaOptions provides Kafka configuration options.
func WithKafkaOptions(opts ...fx_kafka_config.Option) Option {
	return func(o *messagingOptions) {
		o.kafkaOpts = append(o.kafkaOpts, opts...)
	}
}

// NewMessagingModule provides messaging functionality: kafka, outbox, consumer, producer.
//
// Options:
//   - WithKafkaOptions: provide Kafka configuration options
//
// Example usage:
//
//	// Production - loads config from koanf
//	messaging.NewMessagingModule()
//
//	// Testing - with static config
//	messaging.NewMessagingModule(
//	    messaging.WithKafkaOptions(fx_kafka_config.WithKafkaConfig(config.Config{...})),
//	)
func NewMessagingModule(opts ...Option) fx.Option {
	cfg := &messagingOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx_kafka_config.NewKafkaConfigModule(cfg.kafkaOpts...),
		fx_producer.NewProducerModule(),
		fx_kafkaproto.NewProtoModule(),
		fx_outbox.NewOutboxModule(),
	)
}
