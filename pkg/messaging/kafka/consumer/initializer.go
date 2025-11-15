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

// Initialize subscribes to the topic and waits until it's ready
func (i *initializer) Initialize(ctx context.Context) error {
	i.log.Info("initializing consumer", zap.String("topic", i.topic))

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

	if err := i.waitUntilReady(ctxWithTimeout); err != nil {
		return err
	}

	// Check assigned partitions
	assignment, err := i.consumer.Assignment()
	if err != nil {
		return fmt.Errorf("failed to get partition assignment: %w", err)
	}

	if len(assignment) == 0 {
		i.log.Warn("consumer has no assigned partitions - likely more consumers than partitions in the group",
			zap.String("topic", i.topic))
	} else {
		partitionIDs := make([]int32, len(assignment))
		for idx, partition := range assignment {
			partitionIDs[idx] = partition.Partition
		}
		i.log.Info("consumer initialized successfully with assigned partitions",
			zap.String("topic", i.topic),
			zap.Int32s("partitions", partitionIDs))
	}

	return nil
}

func (i *initializer) subscribe() error {
	i.log.Info("subscribing to topic", zap.String("topic", i.topic))

	err := i.consumer.SubscribeTopics([]string{i.topic}, nil)
	if err != nil {
		i.log.Error("failed to subscribe to topic",
			zap.String("topic", i.topic),
			zap.Error(err))
		return err
	}

	i.log.Info("subscribed to topic")
	return nil
}

func (i *initializer) waitUntilReady(ctx context.Context) error {
	i.log.Info("waiting for topic to be ready",
		zap.Int("timeout_seconds", i.timeoutSeconds),
		zap.Bool("fail_on_topic_error", i.failOnTopicError))

	var lastErr error

	for {
		select {
		case <-ctx.Done():
			if i.failOnTopicError {
				if lastErr != nil {
					return fmt.Errorf("%w: %v", ctx.Err(), lastErr)
				}
				return ctx.Err()
			}
			i.log.Warn("timeout waiting for topic, continuing anyway",
				zap.Error(lastErr))
			return nil
		default:
		}

		// Calculate timeout for GetMetadata from context
		timeout := 5 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < timeout {
				timeout = remaining
			}
		}

		// Get topic metadata (allTopics=false means only this specific topic)
		metadata, err := i.consumer.GetMetadata(&i.topic, false, int(timeout.Milliseconds()))
		if err != nil {
			lastErr = err
			i.log.Warn("failed to get topic metadata, retrying",
				zap.String("topic", i.topic),
				zap.Error(err))
			sleep(ctx, 5*time.Second)
			continue
		}

		// Check if topic exists
		topicMeta, ok := metadata.Topics[i.topic]
		if !ok {
			lastErr = fmt.Errorf("topic not found in metadata")
			i.log.Warn("topic not found in metadata, retrying",
				zap.String("topic", i.topic))
			sleep(ctx, 5*time.Second)
			continue
		}

		// Check for topic-level errors
		if topicMeta.Error.Code() != kafka.ErrNoError {
			lastErr = topicMeta.Error
			i.log.Warn("topic has error, retrying",
				zap.String("topic", i.topic),
				zap.String("error", topicMeta.Error.String()))
			sleep(ctx, 5*time.Second)
			continue
		}

		// Check if topic has partitions
		if len(topicMeta.Partitions) == 0 {
			lastErr = fmt.Errorf("topic has no partitions")
			i.log.Warn("topic has no partitions, retrying",
				zap.String("topic", i.topic))
			sleep(ctx, 5*time.Second)
			continue
		}

		i.log.Info("topic is ready",
			zap.String("topic", i.topic),
			zap.Int("partitions", len(topicMeta.Partitions)))
		return nil
	}
}
