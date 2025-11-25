package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
	resultHandler *resultHandler,
	retryExecutor RetryExecutor,
	tracer MessageTracer,
) *processor {
	processor := newProcessor(
		kafkaConsumer,
		messagesChan,
		handler,
		deserializer,
		logger.Logger,
		resultHandler,
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

func provideReader(
	lc fx.Lifecycle,
	_ *initializer,
	kafkaConsumer *kafka.Consumer,
	consumerConf config.ConsumerConfig,
	messagesChan chan *kafka.Message,
	logger consumerLogger,
	readinessWaiter health.ReadinessWaiter,
) *reader {
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
