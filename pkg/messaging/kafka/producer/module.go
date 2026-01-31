package producer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewProducerModule provides Kafka producer components for dependency injection.
func NewProducerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideKafkaProducer,
			provideProducer,
		),
		fx.Invoke(invokeInitializer),
	)
}

func provideKafkaProducer(lc fx.Lifecycle, conf config.Config) (*kafka.Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": conf.Brokers})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			p.Close()
			return nil
		},
	})

	return p, nil
}

func invokeInitializer(lc fx.Lifecycle, readiness health.ComponentManager, p *kafka.Producer, log *zap.Logger, conf config.Config) {
	markReady := readiness.AddComponent("kafka-producer")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := waitForBrokers(ctx, p, log.With(zap.String("component", "producer")), conf.ProducerConfig.ReadinessTimeoutSeconds, conf.ProducerConfig.FailOnBrokerError); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}

func provideProducer(kafkaProducer *kafka.Producer, log *zap.Logger) Producer {
	return newProducer(kafkaProducer, log.With(zap.String("component", "producer")))
}
