package consumer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
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
		fx.Provide(
			fx.Annotate(
				func(conf config.Config) (config.ConsumerConfig, error) {
					return getConsumerConfig(conf, consumerName)
				},
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

func provideInitializer(
	lc fx.Lifecycle,
	consumer *kafka.Consumer,
	consumerConf config.ConsumerConfig,
	logger consumerLogger,
	componentMgr health.ComponentManager,
) *initializer {
	initializer := newInitializer(
		consumer,
		consumerConf.Topic,
		logger.Logger,
		consumerConf.ReadinessTimeoutSeconds,
		consumerConf.FailOnTopicError,
	)
	componentMgr.AddComponent("kafka-consumer-" + consumerConf.Name)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := initializer.initialize(ctx); err != nil {
				return err
			}
			componentMgr.MarkReady("kafka-consumer-" + consumerConf.Name)
			return nil
		},
	})
	return initializer
}

func provideProcessor(
	lc fx.Lifecycle,
	_ *initializer,
	kafkaConsumer *kafka.Consumer,
	messagesChan chan *kafka.Message,
	handler Handler,
	deserializer Deserializer,
	logger consumerLogger,
	dlqHandler DLQHandler,
	retryExecutor RetryExecutor,
	tracer MessageTracer,
) *processor {
	processor := newProcessor(
		kafkaConsumer,
		messagesChan,
		handler,
		deserializer,
		logger.Logger,
		dlqHandler,
		retryExecutor,
		tracer,
	)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			processor.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			processor.stop()
			return nil
		},
	})
	return processor
}

func provideReader(lc fx.Lifecycle,
	_ *initializer,
	kafkaConsumer *kafka.Consumer,
	consumerConf config.ConsumerConfig,
	messagesChan chan *kafka.Message,
	logger consumerLogger,
	readinessWaiter health.ReadinessWaiter) *reader {
	reader := newReader(kafkaConsumer, consumerConf.Topic, messagesChan, logger.Logger, readinessWaiter)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			reader.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			reader.stop()
			return nil
		},
	})
	return reader
}

func provideKafkaConsumer(lc fx.Lifecycle, brokers string, consumerConf config.ConsumerConfig, log *zap.Logger) (*kafka.Consumer, error) {
	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"group.id":                 consumerConf.GroupID,
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
		"auto.commit.interval.ms":  3000,
		"auto.offset.reset":        consumerConf.AutoOffsetReset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer, name: %s: %w", consumerConf.Name, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info("closing kafka consumer")
			return kafkaConsumer.Close()
		},
	})

	return kafkaConsumer, nil
}
