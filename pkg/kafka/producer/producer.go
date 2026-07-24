package producer

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer is the interface for sending messages to Kafka.
type Producer interface {
	Produce(ctx context.Context, record *kgo.Record, promise func(*kgo.Record, error))
}
