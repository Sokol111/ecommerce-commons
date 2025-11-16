package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewConsumerModule() fx.Option {
	return fx.Options(
		fx.Provide(newAvroDeserializer),
		fx.Invoke(func(in kafkaConsumersGroup, log *zap.Logger) {
			log.Info("kafka consumers initialized", zap.Int("count", len(in.Consumers)))
		}),
	)
}

type kafkaConsumersGroup struct {
	fx.In
	Consumers []Consumer `group:"kafka_consumers"`
}

func RegisterHandlerAndConsumer(
	consumerName string,
	handlerConstructor any,
) fx.Option {
	return fx.Module(
		consumerName, // Unique module name
		fx.Provide(
			fx.Annotate(
				handlerConstructor,
				fx.As(new(Handler)),
			),
			fx.Private,
		), // Handler is private to this module
		fx.Provide(
			fx.Annotate(
				func(lc fx.Lifecycle, log *zap.Logger, conf config.Config, h Handler, readiness health.Readiness, deserializer Deserializer, p producer.Producer) (Consumer, error) {
					return provideConsumer(lc, log, conf, consumerName, h, deserializer, readiness, p)
				},
				fx.ResultTags(`group:"kafka_consumers"`), // Consumer is exported to group
			),
		),
	)
}

func provideConsumer(
	lc fx.Lifecycle,
	log *zap.Logger,
	conf config.Config,
	consumerName string,
	handler Handler,
	deserializer Deserializer,
	readiness health.Readiness,
	dlqProducer producer.Producer,
) (Consumer, error) {
	var consumerConf *config.ConsumerConfig

	for _, c := range conf.ConsumersConfig.ConsumerConfig {
		if c.Name == consumerName {
			consumerConf = &c
			break
		}
	}
	if consumerConf == nil {
		return nil, fmt.Errorf("no consumer config found for handler with consumer name: %s", consumerName)
	}

	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        conf.Brokers,
		"group.id":                 consumerConf.GroupID,
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
		"auto.commit.interval.ms":  3000,
		"auto.offset.reset":        consumerConf.AutoOffsetReset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer, topic: %s: %w", consumerConf.Topic, err)
	}

	messagesChan := make(chan *kafka.Message, 100)

	logFields := []zap.Field{
		zap.String("component", "consumer"),
		zap.String("consumer_name", consumerConf.Name),
		zap.String("topic", consumerConf.Topic),
		zap.String("group_id", consumerConf.GroupID),
	}

	// Configure DLQ if enabled
	var dlqTopic string
	var producerForDLQ producer.Producer
	if consumerConf.EnableDLQ {
		dlqTopic = consumerConf.DLQTopic
		producerForDLQ = dlqProducer
		log.Info("DLQ enabled",
			zap.String("consumer_name", consumerConf.Name),
			zap.String("dlq_topic", dlqTopic))
		logFields = append(logFields, zap.String("dlq_topic", dlqTopic))
	}

	reader := newReader(kafkaConsumer, consumerConf.Topic, messagesChan, log.With(logFields...), readiness)
	processor := newProcessor(kafkaConsumer, messagesChan, handler, deserializer, log.With(logFields...), producerForDLQ, dlqTopic)
	initializer := newInitializer(
		kafkaConsumer,
		consumerConf.Topic,
		log.With(logFields...),
		consumerConf.ReadinessTimeoutSeconds,
		consumerConf.FailOnTopicError,
	)

	c := newConsumer(reader, processor, initializer, log.With(logFields...))

	readiness.AddComponent("kafka-consumer-" + consumerName)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := c.Start(ctx); err != nil {
				return err
			}
			// Signal readiness after successful consumer start
			readiness.MarkReady("kafka-consumer-" + consumerName)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return c.Stop(ctx)
		},
	})

	return c, nil
}
