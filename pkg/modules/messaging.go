package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/patterns/outbox"
	"go.uber.org/fx"
)

// NewMessagingModule provides messaging functionality: kafka, outbox, consumer, producer
func NewMessagingModule() fx.Option {
	return fx.Options(
		config.NewKafkaConfigModule(),
		producer.NewProducerModule(),
		avro.NewAvroModule(),
		outbox.NewOutboxModule(),
	)
}
