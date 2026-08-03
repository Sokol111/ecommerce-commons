package fxconfig

import (
	fx_kafka_config "github.com/Sokol111/ecommerce-commons/pkg/kafka/config/fxconfig"
	fx_kafkaproto "github.com/Sokol111/ecommerce-commons/pkg/kafka/kafkaproto/fxconfig"
	fx_outbox "github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox/fxconfig"
	fx_producer "github.com/Sokol111/ecommerce-commons/pkg/kafka/producer/fxconfig"
	"go.uber.org/fx"
)

// NewKafkaModule provides kafka functionality: outbox, consumer, producer.
func NewKafkaModule() fx.Option {
	return fx.Options(
		fx_kafka_config.NewKafkaConfigModule(),
		fx_producer.NewProducerModule(),
		fx_kafkaproto.NewProtoModule(),
		fx_outbox.NewOutboxModule(),
	)
}
