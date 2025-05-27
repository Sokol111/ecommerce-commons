package producer

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ProducerModule = fx.Options(
	fx.Provide(
		provideNewProducer,
	),
)

func provideNewProducer(lc fx.Lifecycle, log *zap.Logger, conf config.Config) (Producer, error) {
	p, err := NewProducer(conf, log)

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
