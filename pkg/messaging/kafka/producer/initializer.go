package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// initializer handles producer initialization: connection readiness check
type initializer struct {
	producer          *kafka.Producer
	log               *zap.Logger
	timeoutSeconds    int  // Timeout for waiting brokers readiness
	failOnBrokerError bool // Whether to fail if brokers are not available
}

func newInitializer(
	producer *kafka.Producer,
	log *zap.Logger,
	timeoutSeconds int,
	failOnBrokerError bool,
) *initializer {
	return &initializer{
		producer:          producer,
		log:               log,
		timeoutSeconds:    timeoutSeconds,
		failOnBrokerError: failOnBrokerError,
	}
}

// Initialize checks if Kafka brokers are ready
func (i *initializer) Initialize(ctx context.Context) error {
	i.log.Info("initializing producer")

	// Wait for brokers to be ready with timeout
	ctxWithTimeout := ctx
	if i.timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctxWithTimeout, cancel = context.WithTimeout(ctx, time.Duration(i.timeoutSeconds)*time.Second)
		defer cancel()
	}

	if err := i.waitUntilReady(ctxWithTimeout); err != nil {
		return err
	}

	i.log.Info("producer initialized successfully")
	return nil
}

func (i *initializer) waitUntilReady(ctx context.Context) error {
	i.log.Info("waiting for kafka brokers to be ready",
		zap.Int("timeout_seconds", i.timeoutSeconds),
		zap.Bool("fail_on_broker_error", i.failOnBrokerError))

	var lastErr error

	for {
		select {
		case <-ctx.Done():
			if i.failOnBrokerError {
				if lastErr != nil {
					return fmt.Errorf("%w: %v", ctx.Err(), lastErr)
				}
				return ctx.Err()
			}
			i.log.Warn("timeout waiting for brokers, continuing anyway",
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

		// Get metadata from brokers (producer has this method directly)
		metadata, err := i.producer.GetMetadata(nil, false, int(timeout.Milliseconds()))
		if err != nil {
			lastErr = err
			i.log.Warn("failed to get metadata from kafka brokers, retrying",
				zap.Error(err))
			sleep(ctx, 5*time.Second)
			continue
		}

		// Check if brokers are available
		if len(metadata.Brokers) == 0 {
			lastErr = fmt.Errorf("no brokers available")
			i.log.Warn("no brokers available, retrying")
			sleep(ctx, 5*time.Second)
			continue
		}

		i.log.Info("kafka brokers are ready",
			zap.Int("brokers_count", len(metadata.Brokers)))
		return nil
	}
}

// sleep waits for the specified duration or until context is cancelled
func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}
