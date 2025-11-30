package producer

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// metadataProvider is the interface for getting Kafka metadata.
type metadataProvider interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
}

func waitForBrokers(ctx context.Context, p metadataProvider, log *zap.Logger, timeoutSec int, failOnError bool) error {
	log.Info("waiting for kafka brokers", zap.Int("timeout_seconds", timeoutSec))

	if timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	if err := pollBrokers(ctx, p); err != nil {
		if failOnError {
			return err
		}
		log.Warn("brokers not ready, continuing", zap.Error(err))
	}

	log.Info("producer ready")
	return nil
}

func pollBrokers(ctx context.Context, p metadataProvider) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if meta, err := p.GetMetadata(nil, false, 5000); err == nil && len(meta.Brokers) > 0 {
			return nil
		}
	}
}
