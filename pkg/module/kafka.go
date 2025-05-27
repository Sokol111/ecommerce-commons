package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/kafka"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"go.uber.org/fx"
)

var KafkaWritingModule = fx.Options(
	kafka.KafkaConfigModule,
	producer.ProducerModule,
	outbox.OutboxModule,
)
