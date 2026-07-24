package fxconfig

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/worker"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/consumer"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/twmb/franz-go/pkg/kgo"
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
				fx.As(new(consumer.Handler)),
			),
			provideConsumerClient,
			consumer.NewProcessor,
			consumer.NewMessageDeserializer,
			consumer.NewMessageTracer,
			consumer.NewResultHandler,
			consumer.NewReader,
			provideMessageChannel,
			provideEnvelopeChannel,
			provideDLQHandler,
			fx.Private,
		),
		fx.Invoke(
			worker.RunWorker[*consumer.Reader]("reader", worker.WithTrafficReady(), worker.WithShutdown()),
			worker.RunWorker[*consumer.MessageDeserializer]("deserializer"),
			worker.RunWorker[*consumer.Processor]("processor"),
		),
	)
}

func provideMessageChannel(consumerConf config.ConsumerConfig) chan *kgo.Record {
	return make(chan *kgo.Record, consumerConf.ChannelBufferSize)
}

func provideEnvelopeChannel(consumerConf config.ConsumerConfig) chan *consumer.MessageEnvelope {
	return make(chan *consumer.MessageEnvelope, consumerConf.ChannelBufferSize)
}

func provideDLQHandler(consumerConf config.ConsumerConfig, tracer consumer.MessageTracer, dlqProducer producer.Producer, logger *zap.Logger) consumer.DLQHandler {
	if consumerConf.EnableDLQ {
		return consumer.NewDLQHandler(dlqProducer, consumerConf.DLQTopic, tracer, logger)
	}
	return consumer.NewNoopDLQHandler(logger)
}
