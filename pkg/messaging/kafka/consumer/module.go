package consumer

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
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

// RegisterHandlerAndConsumer creates a Kafka consumer module with the specified handler.
func RegisterHandlerAndConsumer(
	consumerName string,
	handlerConstructor any,
) fx.Option {
	return fx.Module(
		consumerName, // Unique module name
		fx.Decorate(
			func(log *zap.Logger, consumerConf config.ConsumerConfig) *zap.Logger {
				return log.With(
					zap.String("component", "consumer"),
					zap.String("consumer_name", consumerConf.Name),
					zap.String("topic", consumerConf.Topic),
					zap.String("group_id", consumerConf.GroupID),
				)
			},
		),
		fx.Supply(
			fx.Annotate(
				consumerName,
				fx.ResultTags(`name:"consumerName"`),
			),
			fx.Private,
		),
		fx.Provide(
			fx.Annotate(
				getConsumerConfig,
				fx.ParamTags(``, `name:"consumerName"`),
			),
			fx.Annotate(
				handlerConstructor,
				fx.As(new(Handler)),
			),
			provideKafkaConsumer,
			newProcessor,
			newMessageDeserializer,
			newMessageTracer,
			newResultHandler,
			newReader,
			provideMessageChannel,
			provideEnvelopeChannel,
			provideDLQHandler,
			fx.Private,
		),
		fx.Invoke(
			worker.RunWorker[*reader]("reader", worker.WithTrafficReady(), worker.WithShutdown()),
			worker.RunWorker[*messageDeserializer]("deserializer"),
			worker.RunWorker[*processor]("processor"),
		),
	)
}

func provideMessageChannel(consumerConf config.ConsumerConfig) chan *kafka.Message {
	return make(chan *kafka.Message, consumerConf.ChannelBufferSize)
}

func provideEnvelopeChannel(consumerConf config.ConsumerConfig) chan *MessageEnvelope {
	return make(chan *MessageEnvelope, consumerConf.ChannelBufferSize)
}

func provideDLQHandler(consumerConf config.ConsumerConfig, tracer MessageTracer, dlqProducer producer.Producer, logger *zap.Logger) DLQHandler {
	if consumerConf.EnableDLQ {
		return newDLQHandler(dlqProducer, consumerConf.DLQTopic, tracer, logger)
	}
	return newNoopDLQHandler(logger)
}
