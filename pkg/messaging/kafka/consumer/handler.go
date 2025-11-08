package consumer

import (
	"context"
	"errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ErrSkipMessage indicates that the message should be skipped without retry.
// Use this when the handler doesn't need to process this particular event type
// or when the message is intentionally ignored.
var ErrSkipMessage = errors.New("skip message processing")

// DeserializerFunc is a legacy function that deserializes raw bytes into a typed event
// based on the message headers (e.g., event-type header).
// It should return ErrSkipMessage if the event type should be ignored.
//
// Deprecated: Use Deserializer interface with Schema Registry instead.
type DeserializerFunc func(data []byte, headers []kafka.Header) (any, error)

type Handler interface {
	Process(ctx context.Context, event any) error
}
