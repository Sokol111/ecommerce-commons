package producer

import (
	"context"
	"strings"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewProducerModule provides Kafka producer components for dependency injection.
func NewProducerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideKgoClient,
			provideProducer,
		),
		fx.Invoke(invokeInitializer),
	)
}

func provideKgoClient(lc fx.Lifecycle, conf config.Config) (*kgo.Client, error) {
	brokers := strings.Split(conf.Brokers, ",")

	compression := compressionCodec(conf.ProducerConfig.Compression)

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerLinger(conf.ProducerConfig.Linger),
		kgo.ProducerBatchCompression(compression),
		kgo.RecordDeliveryTimeout(conf.ProducerConfig.DeliveryTimeout),
		kgo.MaxBufferedRecords(conf.ProducerConfig.MaxBufferedRecords),
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	})

	return client, nil
}

func invokeInitializer(lc fx.Lifecycle, readiness health.ComponentManager, client *kgo.Client, log *zap.Logger, conf config.Config) {
	markReady := readiness.AddComponent("kafka-producer")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := waitForBrokers(ctx, client, log.With(zap.String("component", "producer")), conf.ProducerConfig.ReadinessTimeoutSeconds, conf.ProducerConfig.FailOnBrokerError); err != nil {
				return err
			}
			markReady()
			return nil
		},
	})
}

func provideProducer(client *kgo.Client) Producer {
	return client
}

func compressionCodec(name string) kgo.CompressionCodec {
	switch name {
	case "snappy":
		return kgo.SnappyCompression()
	case "lz4":
		return kgo.Lz4Compression()
	case "zstd":
		return kgo.ZstdCompression()
	case "none":
		return kgo.NoCompression()
	default:
		return kgo.NoCompression()
	}
}
