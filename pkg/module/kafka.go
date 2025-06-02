package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/consumer"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"go.uber.org/fx"
)

func NewKafkaModule() fx.Option {
	return fx.Options(
		config.NewKafkaConfigModule(),
		producer.NewProducerModule(),
		outbox.NewOutboxModule(),
		consumer.NewConsumerModule(),
	)
}
