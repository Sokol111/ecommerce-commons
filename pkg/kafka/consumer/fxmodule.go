package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type HandlerDef struct {
	Name    string
	Handler Handler[any]
}

type handlersGroup struct {
	fx.In
	Handlers []HandlerDef `group:"kafka_handlers"`
}

var ConsumerModule = fx.Options(
	fx.Provide(
		provideConsumers,
	),
)

func provideConsumers(lc fx.Lifecycle, log *zap.Logger, conf kafka.Config, group handlersGroup) ([]Consumer, error) {
	handlersMap := make(map[string]Handler[any])
	for _, h := range group.Handlers {
		handlersMap[h.Name] = h.Handler
	}

	var consumers []Consumer
	for _, consumerConf := range conf.Consumers {
		handler, ok := handlersMap[consumerConf.Handler]
		if !ok {
			return nil, fmt.Errorf("no handler found for topic: %s", consumerConf.Topic)
		}
		c, err := provideNewConsumer(lc, log, conf.Brokers, consumerConf, handler)
		if err != nil {
			return nil, err
		}
		consumers = append(consumers, c)
	}

	return consumers, nil
}

func provideNewConsumer(lc fx.Lifecycle, log *zap.Logger, brokers string, consumerConf kafka.ConsumerConfig, handler Handler[any]) (Consumer, error) {
	c, err := NewConsumer(brokers, consumerConf.GroupID, consumerConf.Topic, consumerConf.AutoOffsetReset, handler, log)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return c.Start()
		},
		OnStop: func(ctx context.Context) error {
			return c.Stop(ctx)
		},
	})

	return c, nil
}
