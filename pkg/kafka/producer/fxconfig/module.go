package fxconfig

import (
	"context"
	"strings"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/twmb/franz-go/pkg/kadm"
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
		kgo.AllowAutoTopicCreation(),
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

func provideProducer(client *kgo.Client) producer.Producer {
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

func waitForBrokers(ctx context.Context, client *kgo.Client, log *zap.Logger, timeoutSec int, failOnError bool) error {
	log.Info("waiting for kafka brokers", zap.Int("timeout_seconds", timeoutSec))

	if timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	if err := pollBrokers(ctx, client); err != nil {
		if failOnError {
			return err
		}
		log.Warn("brokers not ready, continuing", zap.Error(err))
	}

	log.Info("producer ready")
	return nil
}

func pollBrokers(ctx context.Context, client *kgo.Client) error {
	admClient := kadm.NewClient(client)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		brokers, err := admClient.ListBrokers(ctx)
		if err == nil && len(brokers) > 0 {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}
