package consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideKafkaConsumer(lc fx.Lifecycle, conf config.Config, consumerConf config.ConsumerConfig, log *zap.Logger, componentMgr health.ComponentManager) (*kafka.Consumer, messageReader, offsetStorer, error) {
	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        conf.Brokers,
		"group.id":                 consumerConf.GroupID,
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
		"auto.commit.interval.ms":  3000,
		"auto.offset.reset":        consumerConf.AutoOffsetReset,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create kafka consumer, name: %s: %w", consumerConf.Name, err)
	}

	componentName := "kafka-consumer-" + consumerConf.Name
	markReady := componentMgr.AddComponent(componentName)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("subscribing to topic", zap.String("topic", consumerConf.Topic))

			rebalanceCb := func(c *kafka.Consumer, event kafka.Event) error {
				switch ev := event.(type) {
				case kafka.AssignedPartitions:
					logPartitionEvent(log, "partitions assigned", ev.Partitions)
				case kafka.RevokedPartitions:
					logPartitionEvent(log, "partitions revoked", ev.Partitions)
				}
				return nil
			}

			if err := kafkaConsumer.SubscribeTopics([]string{consumerConf.Topic}, rebalanceCb); err != nil {
				log.Error("failed to subscribe to topic", zap.Error(err))
				return err
			}

			// Verify topic is available
			if err := verifyTopicAvailable(kafkaConsumer, consumerConf.Topic, log); err != nil {
				if consumerConf.FailOnTopicError {
					return err
				}
				log.Warn("topic verification failed, continuing anyway", zap.Error(err))
			}

			markReady()
			return nil
		},
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

	return kafkaConsumer, kafkaConsumer, kafkaConsumer, nil
}

func logPartitionEvent(log *zap.Logger, event string, partitions []kafka.TopicPartition) {
	if len(partitions) == 0 {
		log.Warn(event + ": no partitions")
		return
	}

	partitionIDs := make([]int32, len(partitions))
	for idx, partition := range partitions {
		partitionIDs[idx] = partition.Partition
	}

	log.Info(event,
		zap.Int("partition_count", len(partitions)),
		zap.Int32s("partitions", partitionIDs))
}

// verifyTopicAvailable checks if topic exists and has partitions.
func verifyTopicAvailable(consumer *kafka.Consumer, topic string, log *zap.Logger) error {
	metadata, err := consumer.GetMetadata(&topic, false, 10000)
	if err != nil {
		return fmt.Errorf("failed to get topic metadata: %w", err)
	}

	topicMeta, ok := metadata.Topics[topic]
	if !ok {
		return fmt.Errorf("topic %s not found in metadata", topic)
	}

	if topicMeta.Error.Code() != kafka.ErrNoError {
		return fmt.Errorf("topic %s has error: %s", topic, topicMeta.Error.String())
	}

	if len(topicMeta.Partitions) == 0 {
		return fmt.Errorf("topic %s has no partitions", topic)
	}

	log.Info("topic is ready",
		zap.String("topic", topic),
		zap.Int("partitions", len(topicMeta.Partitions)))
	return nil
}
