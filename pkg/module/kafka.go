package module

import (
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/outbox"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"go.uber.org/fx"
)

var KafkaWritingModule = fx.Options(
	config.KafkaConfigModule,
	producer.ProducerModule,
	outbox.OutboxModule,
)
