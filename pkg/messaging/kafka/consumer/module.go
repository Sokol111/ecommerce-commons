package consumer

import (
	"fmt"
	"reflect"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func getConsumerConfig(conf config.Config, consumerName string) (config.ConsumerConfig, error) {
	for _, c := range conf.ConsumersConfig.ConsumerConfig {
		if c.Name == consumerName {
			return c, nil
		}
	}
	return config.ConsumerConfig{}, fmt.Errorf("no consumer config found for consumer name: %s", consumerName)
}

type consumerLogger struct {
	*zap.Logger
}

func RegisterHandlerAndConsumer(
	consumerName string,
	handlerConstructor any,
	typeMappingParam map[string]reflect.Type,
) fx.Option {
	return fx.Module(
		consumerName, // Unique module name
		fx.Supply(
			fx.Annotate(
				consumerName,
				fx.ResultTags(`name:"consumerName"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				getConsumerConfig,
				fx.ParamTags(``, `name:"consumerName"`),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(lc fx.Lifecycle, conf config.Config, consumerConf config.ConsumerConfig, logger consumerLogger) (*kafka.Consumer, error) {
					return provideKafkaConsumer(lc, conf.Brokers, consumerConf, logger.Logger)
				},
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				handlerConstructor,
				fx.As(new(Handler)),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func() typeMapping { return typeMappingParam },
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				newRegistrySchemaResolver,
				fx.As(new(SchemaResolver)),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				newAvroDeserializer,
				fx.As(new(Deserializer)),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				provideProcessor,
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				provideInitializer,
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(consumerConf config.ConsumerConfig, logger consumerLogger) RetryExecutor {
					return newRetryExecutor(consumerConf.MaxRetryAttempts, consumerConf.InitialBackoff, consumerConf.MaxBackoff, logger.Logger)
				},
				fx.As(new(RetryExecutor)),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				newMessageTracer,
				fx.As(new(MessageTracer)),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(consumerConf config.ConsumerConfig) chan *kafka.Message {
					return make(chan *kafka.Message, consumerConf.ChannelBufferSize)
				},
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(consumerConf config.ConsumerConfig, tracer MessageTracer, dlqProducer producer.Producer, logger consumerLogger) DLQHandler {
					if consumerConf.EnableDLQ {
						return newDLQHandler(dlqProducer, consumerConf.DLQTopic, tracer, logger.Logger)
					}
					return newNoopDLQHandler(logger.Logger)
				},
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(logger consumerLogger, dlqHandler DLQHandler, consumer *kafka.Consumer) *resultHandler {
					return newResultHandler(logger.Logger, dlqHandler, consumer)
				},
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				func(
					log *zap.Logger,
					consumerConf config.ConsumerConfig,
				) consumerLogger {
					return consumerLogger{
						Logger: log.With(
							zap.String("component", "consumer"),
							zap.String("consumer_name", consumerConf.Name),
							zap.String("topic", consumerConf.Topic),
							zap.String("group_id", consumerConf.GroupID),
						),
					}
				},
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				provideReader,
			),
			fx.Private,
		),
	)
}
