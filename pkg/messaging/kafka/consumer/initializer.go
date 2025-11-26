package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// initializer handles consumer initialization: subscription and readiness check
type initializer struct {
	consumer         *kafka.Consumer
	topic            string
	log              *zap.Logger
	timeoutSeconds   int  // Timeout for waiting topic readiness
	failOnTopicError bool // Whether to fail if topic is not available
}

func newInitializer(
	consumer *kafka.Consumer,
	topic string,
	log *zap.Logger,
	timeoutSeconds int,
	failOnTopicError bool,
) *initializer {
	return &initializer{
		consumer:         consumer,
		topic:            topic,
		log:              log,
		timeoutSeconds:   timeoutSeconds,
		failOnTopicError: failOnTopicError,
	}
}

// initialize subscribes to the topic and waits until it's ready
func (i *initializer) initialize(ctx context.Context) error {
	i.log.Info("initializing consumer")

	// Step 1: Subscribe to topic
	if err := i.subscribe(); err != nil {
		return err
	}

	// Step 2: Wait for topic to be ready with timeout
	ctxWithTimeout := ctx
	if i.timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctxWithTimeout, cancel = context.WithTimeout(ctx, time.Duration(i.timeoutSeconds)*time.Second)
		defer cancel()
	}

	i.log.Info("waiting for topic to be ready",
		zap.Int("timeout_seconds", i.timeoutSeconds),
		zap.Bool("fail_on_topic_error", i.failOnTopicError))

	err := i.waitUntilReady(ctxWithTimeout)

	if err != nil {
		if i.failOnTopicError {
			return err
		}
		i.log.Warn("timeout waiting for topic, continuing anyway", zap.Error(err))
	}

	return nil
}

func (i *initializer) subscribe() error {
	i.log.Info("subscribing to topic")

	rebalanceCb := func(c *kafka.Consumer, event kafka.Event) error {
		switch ev := event.(type) {
		case kafka.AssignedPartitions:
			i.logPartitionEvent("partitions assigned", ev.Partitions)
		case kafka.RevokedPartitions:
			i.logPartitionEvent("partitions revoked", ev.Partitions)
		}
		return nil
	}

	err := i.consumer.SubscribeTopics([]string{i.topic}, rebalanceCb)
	if err != nil {
		i.log.Error("failed to subscribe to topic", zap.Error(err))
		return err
	}

	return nil
}

func (i *initializer) logPartitionEvent(event string, partitions []kafka.TopicPartition) {
	if len(partitions) == 0 {
		i.log.Warn(event + ": no partitions")
		return
	}

	partitionIDs := make([]int32, len(partitions))
	for idx, partition := range partitions {
		partitionIDs[idx] = partition.Partition
	}

	i.log.Info(event,
		zap.Int("partition_count", len(partitions)),
		zap.Int32s("partitions", partitionIDs))
}

func (i *initializer) waitUntilReady(ctx context.Context) error {
	var lastErr error
	var lastLogTime time.Time

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w: %v", ctx.Err(), lastErr)
			}
			return ctx.Err()
		default:
		}

		// Calculate timeout for GetMetadata from context
		timeout := 10 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < timeout {
				timeout = remaining
			}
		}

		// Get topic metadata (allTopics=false means only this specific topic)
		metadata, err := i.consumer.GetMetadata(&i.topic, false, int(timeout.Milliseconds()))
		if err != nil {
			if lastErr == nil || time.Since(lastLogTime) > 30*time.Second {
				lastLogTime = time.Now()
				i.log.Debug("failed to get topic metadata, retrying", zap.Error(err))
			}
			lastErr = err
			continue
		}

		// Check if topic exists
		topicMeta, ok := metadata.Topics[i.topic]
		if !ok {
			err := fmt.Errorf("topic not found in metadata")
			if lastErr == nil || time.Since(lastLogTime) > 30*time.Second {
				lastLogTime = time.Now()
				i.log.Debug("topic not found in metadata, retrying")
			}
			lastErr = err
			continue
		}

		// Check for topic-level errors
		if topicMeta.Error.Code() != kafka.ErrNoError {
			err := topicMeta.Error
			if lastErr == nil || time.Since(lastLogTime) > 30*time.Second {
				lastLogTime = time.Now()
				i.log.Debug("topic has error, retrying", zap.String("error", topicMeta.Error.String()))
			}
			lastErr = err
			continue
		}

		// Check if topic has partitions
		if len(topicMeta.Partitions) == 0 {
			err := fmt.Errorf("topic has no partitions")
			if lastErr == nil || time.Since(lastLogTime) > 30*time.Second {
				lastLogTime = time.Now()
				i.log.Debug("topic has no partitions, retrying")
			}
			lastErr = err
			continue
		}

		i.log.Info("topic is ready", zap.Int("partitions", len(topicMeta.Partitions)))
		return nil
	}
}
