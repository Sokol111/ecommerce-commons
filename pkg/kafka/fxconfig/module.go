package fxconfig

import (
	fx_kafka_config "github.com/Sokol111/ecommerce-commons/pkg/kafka/config/fxconfig"
	fx_kafkaproto "github.com/Sokol111/ecommerce-commons/pkg/kafka/kafkaproto/fxconfig"
	fx_outbox "github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox/fxconfig"
	fx_producer "github.com/Sokol111/ecommerce-commons/pkg/kafka/producer/fxconfig"
	"go.uber.org/fx"
)

// kafkaOptions holds internal configuration for the Kafka module.
type kafkaOptions struct {
	kafkaOpts []fx_kafka_config.Option
}

// Option is a functional option for configuring the Kafka module.
type Option func(*kafkaOptions)

// WithKafkaOptions provides Kafka configuration options.
func WithKafkaOptions(opts ...fx_kafka_config.Option) Option {
	return func(o *kafkaOptions) {
		o.kafkaOpts = append(o.kafkaOpts, opts...)
	}
}

// NewKafkaModule provides kafka functionality: outbox, consumer, producer.
//
// Options:
//   - WithKafkaOptions: provide Kafka configuration options
//
// Example usage:
//
//	// Production - loads config from koanf
//	kafka.NewKafkaModule()
//
//	// Testing - with static config
//	kafka.NewKafkaModule(
//	    kafka.WithKafkaOptions(fx_kafka_config.WithKafkaConfig(config.Config{...})),
//	)
func NewKafkaModule(opts ...Option) fx.Option {
	cfg := &kafkaOptions{}
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
