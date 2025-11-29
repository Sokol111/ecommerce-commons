package consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideInitializer(
	lc fx.Lifecycle,
	consumer *kafka.Consumer,
	consumerConf config.ConsumerConfig,
	logger *zap.Logger,
	componentMgr health.ComponentManager,
) *initializer {
	initializer := newInitializer(
		consumer,
		consumerConf.Topic,
		logger,
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

func provideKafkaConsumer(lc fx.Lifecycle, conf config.Config, consumerConf config.ConsumerConfig, log *zap.Logger) (*kafka.Consumer, error) {
	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        conf.Brokers,
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
			// Final commit before closing
			if _, commitErr := kafkaConsumer.Commit(); commitErr != nil {
				var kafkaErr kafka.Error
				if !errors.As(commitErr, &kafkaErr) || kafkaErr.Code() != kafka.ErrNoOffset {
					log.Warn("failed to commit offsets on shutdown", zap.Error(commitErr))
				}
			} else {
				log.Debug("final commit successful")
			}

			log.Info("closing kafka consumer")
			return kafkaConsumer.Close()
		},
	})

	return kafkaConsumer, nil
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
