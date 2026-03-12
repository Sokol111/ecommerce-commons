package consumer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideConsumerClient(lc fx.Lifecycle, conf config.Config, consumerConf config.ConsumerConfig, log *zap.Logger, componentMgr health.ComponentManager) (*kgo.Client, error) {
	brokers := strings.Split(conf.Brokers, ",")

	resetOffset := kgo.NewOffset().AtEnd()
	if consumerConf.AutoOffsetReset == "earliest" {
		resetOffset = kgo.NewOffset().AtStart()
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerConf.GroupID),
		kgo.ConsumeTopics(consumerConf.Topic),
		kgo.ConsumeResetOffset(resetOffset),
		kgo.AutoCommitInterval(3 * time.Second),
		kgo.AutoCommitMarks(),
		kgo.OnPartitionsAssigned(func(ctx context.Context, cl *kgo.Client, assigned map[string][]int32) {
			for topic, parts := range assigned {
				log.Info("partitions assigned",
					zap.String("topic", topic),
					zap.Int("partition_count", len(parts)),
					zap.Int32s("partitions", parts))
			}
		}),
		kgo.OnPartitionsRevoked(func(ctx context.Context, cl *kgo.Client, revoked map[string][]int32) {
			for topic, parts := range revoked {
				log.Info("partitions revoked",
					zap.String("topic", topic),
					zap.Int("partition_count", len(parts)),
					zap.Int32s("partitions", parts))
			}
		}),
		kgo.OnPartitionsLost(func(ctx context.Context, cl *kgo.Client, lost map[string][]int32) {
			for topic, parts := range lost {
				log.Warn("partitions lost",
					zap.String("topic", topic),
					zap.Int("partition_count", len(parts)),
					zap.Int32s("partitions", parts))
			}
		}),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer, name: %s: %w", consumerConf.Name, err)
	}

	componentName := "kafka-consumer-" + consumerConf.Name
	markReady := componentMgr.AddComponent(componentName)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("subscribing to topic", zap.String("topic", consumerConf.Topic))

			// Verify topic is available
			if err := verifyTopicAvailable(ctx, client, consumerConf.Topic, log); err != nil {
				if consumerConf.FailOnTopicError {
					return err
				}
				log.Warn("topic verification failed, continuing anyway", zap.Error(err))
			}

			markReady()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("closing kafka consumer")
			client.Close()
			return nil
		},
	})

	return client, nil
}

// verifyTopicAvailable checks if topic exists and has partitions.
func verifyTopicAvailable(ctx context.Context, client *kgo.Client, topic string, log *zap.Logger) error {
	admClient := kadm.NewClient(client)
	topics, err := admClient.ListTopics(ctx, topic)
	if err != nil {
		return fmt.Errorf("failed to get topic metadata: %w", err)
	}

	topicDetail, ok := topics[topic]
	if !ok {
		return fmt.Errorf("topic %s not found in metadata", topic)
	}

	if topicDetail.Err != nil {
		return fmt.Errorf("topic %s has error: %w", topic, topicDetail.Err)
	}

	if len(topicDetail.Partitions) == 0 {
		return fmt.Errorf("topic %s has no partitions", topic)
	}

	log.Info("topic is ready",
		zap.String("topic", topic),
		zap.Int("partitions", len(topicDetail.Partitions)))
	return nil
}
