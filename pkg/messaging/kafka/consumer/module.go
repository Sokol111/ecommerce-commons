package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewConsumerModule() fx.Option {
	return fx.Invoke(func(in kafkaConsumersGroup, log *zap.Logger) {
		log.Info("kafka consumers initialized", zap.Int("count", len(in.Consumers)))
	})
}

type kafkaConsumersGroup struct {
	fx.In
	Consumers []Consumer `group:"kafka_consumers"`
}

func RegisterHandlerAndConsumer(
	consumerName string,
	handlerConstructor any,
	unmarshal UnmarshalFunc,
) fx.Option {
	return fx.Module(
		consumerName, // Unique module name
		fx.Provide(handlerConstructor, fx.Private), // Handler is private to this module
		fx.Provide(
			fx.Annotate(
				func(lc fx.Lifecycle, log *zap.Logger, conf config.Config, h Handler, readiness health.Readiness) (Consumer, error) {
					return provideConsumer(lc, log, conf, consumerName, h, unmarshal, readiness)
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
	unmarshal UnmarshalFunc,
	readiness health.Readiness,
) (Consumer, error) {
	// Create deserializer from unmarshal function
	deserializer := NewDeserializer(unmarshal)

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

	groupId := consumerConf.GroupID
	if groupId == "" {
		groupId = conf.ConsumersConfig.GroupID
	}
	autoOffsetReset := consumerConf.AutoOffsetReset
	if autoOffsetReset == "" {
		autoOffsetReset = conf.ConsumersConfig.AutoOffsetReset
	}

	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        conf.Brokers,
		"group.id":                 groupId,
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
		"auto.commit.interval.ms":  3000,
		"auto.offset.reset":        autoOffsetReset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer, topic: %s: %w", consumerConf.Topic, err)
	}

	messagesChan := make(chan *kafka.Message, 100)

	logFields := []zap.Field{
		zap.String("component", "consumer"),
		zap.String("consumer_name", consumerConf.Name),
		zap.String("topic", consumerConf.Topic),
		zap.String("group_id", groupId),
	}

	reader := newReader(kafkaConsumer, consumerConf.Topic, messagesChan, log.With(logFields...))
	processor := newProcessor(kafkaConsumer, messagesChan, handler, deserializer, log.With(logFields...))

	c := newConsumer(reader, processor, log.With(logFields...))

	readiness.AddOne()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			defer readiness.Done()
			return c.Start()
		},
		OnStop: func(ctx context.Context) error {
			return c.Stop(ctx)
		},
	})

	return c, nil
}
