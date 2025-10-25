package producer

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewProducerModule() fx.Option {
	return fx.Provide(
		provideProducer,
	)
}

func provideProducer(lc fx.Lifecycle, log *zap.Logger, conf config.Config) (Producer, error) {
	p, err := newProducer(conf, log)

	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			p.Close()
			return nil
		},
	})

	return p, nil
}
