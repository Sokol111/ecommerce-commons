package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/consumer"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"go.uber.org/fx"
)

var KafkaModule = fx.Options(
	config.KafkaConfigModule,
	producer.ProducerModule,
	outbox.OutboxModule,
	consumer.ConsumerModule,
)
