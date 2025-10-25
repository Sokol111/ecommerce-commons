package consumer

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewConsumerModule() fx.Option {
	return fx.Invoke(func(in kafkaConsumersGroup, log *zap.Logger) {
		log.Info("kafka consumers initialized", zap.Int("len", len(in.Consumers)))
	})
}

type kafkaConsumersGroup struct {
	fx.In
	Consumers []Consumer `group:"kafka_consumers"`
}

type handlerDef[T any] struct {
	Name    string
	Handler Handler[T]
}

func RegisterHandlerAndConsumer[T any](
	name string,
	constructor any,
) fx.Option {
	return fx.Options(
		fx.Provide(
			constructor,
			func(h Handler[T]) handlerDef[T] {
				return handlerDef[T]{Name: name, Handler: h}
			},
		),
		fx.Provide(
			fx.Annotate(
				provideNewConsumer[T],
				fx.ResultTags(`group:"kafka_consumers"`),
			),
		),
	)
}

func provideNewConsumer[T any](lc fx.Lifecycle, log *zap.Logger, conf config.Config, handlerDef handlerDef[T]) (Consumer, error) {
	var consumerConf *config.ConsumerConfig

	for _, c := range conf.ConsumersConfig.ConsumerConfig {
		if c.Handler == handlerDef.Name {
			consumerConf = &c
			break
		}
	}
	if consumerConf == nil {
		return nil, fmt.Errorf("no consumer config found for handler: %s", handlerDef.Name)
	}
	groupId := consumerConf.GroupID
	if groupId == "" {
		groupId = conf.ConsumersConfig.GroupID
	}
	autoOffsetReset := consumerConf.AutoOffsetReset
	if autoOffsetReset == "" {
		autoOffsetReset = conf.ConsumersConfig.AutoOffsetReset
	}
	c, err := newConsumer(conf.Brokers, groupId, consumerConf.Topic, autoOffsetReset, handlerDef.Handler, log)
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
