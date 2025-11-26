package producer

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewProducerModule() fx.Option {
	return fx.Provide(
		provideProducer,
	)
}

func provideProducer(lc fx.Lifecycle, log *zap.Logger, conf config.Config, readiness health.ComponentManager) (Producer, error) {
	p, err := newProducer(conf, log.With(zap.String("component", "producer")))

	if err != nil {
		return nil, err
	}

	// Create initializer separately
	init := newInitializer(
		p.(*producer).producer,
		log.With(zap.String("component", "producer")),
		conf.ProducerConfig.ReadinessTimeoutSeconds,
		*conf.ProducerConfig.FailOnBrokerError,
	)

	readiness.AddComponent("kafka-producer")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := init.initialize(ctx); err != nil {
				return err
			}
			// Signal readiness after successful producer initialization
			readiness.MarkReady("kafka-producer")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Close()
			return nil
		},
	})

	return p, nil
}
