package consumer

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
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
			provideProcessor,
			provideMessageDeserializer,
			provideInitializer,
			newRetryExecutor,
			newMessageTracer,
			newResultHandler,
			provideReader,
			provideMessageChannel,
			provideEnvelopeChannel,
			provideDLQHandler,
			fx.Private,
		),
		fx.Invoke(func(*initializer, *messageDeserializer, *processor, *reader) {}),
	)
}
