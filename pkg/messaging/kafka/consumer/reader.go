package consumer

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
)

type reader struct {
	client         *kgo.Client
	messagesChan   chan<- *kgo.Record
	maxPollRecords int
	log            *zap.Logger
	throttler      *logger.LogThrottler
}

func newReader(
	client *kgo.Client,
	messagesChan chan *kgo.Record,
	consumerConf config.ConsumerConfig,
	log *zap.Logger,
) *reader {
	return &reader{
		client:         client,
		messagesChan:   messagesChan,
		maxPollRecords: consumerConf.MaxPollRecords,
		log:            log,
		throttler:      logger.NewLogThrottler(log, 0),
	}
}

func (r *reader) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fetches := r.client.PollRecords(ctx, r.maxPollRecords)
		if ctx.Err() != nil {
			return nil //nolint:nilerr // context cancellation is a graceful shutdown, not an error
		}

		fetchErrors := fetches.Errors()
		for _, fe := range fetchErrors {
			if errors.Is(fe.Err, context.Canceled) || errors.Is(fe.Err, context.DeadlineExceeded) {
				continue
			}

			if errors.Is(fe.Err, kerr.UnknownTopicOrPartition) {
				r.throttler.Warn("topic_not_available", "topic not available, waiting for topic creation",
					zap.String("topic", fe.Topic),
					zap.Error(fe.Err))
				continue
			}

			r.log.Error("kafka fetch error",
				zap.String("topic", fe.Topic),
				zap.Int32("partition", fe.Partition),
				zap.Error(fe.Err))
		}

		fetches.EachRecord(func(record *kgo.Record) {
			select {
			case <-ctx.Done():
				return
			case r.messagesChan <- record:
			}
		})
	}
}
