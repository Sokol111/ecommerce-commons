package producer

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

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
