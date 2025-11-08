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

// Deserializer is a function that deserializes raw bytes into a typed event
// based on the message headers (e.g., event-type header).
// It should return ErrSkipMessage if the event type should be ignored.
type Deserializer func(data []byte, headers []kafka.Header) (any, error)

type Handler interface {
	Process(ctx context.Context, event any) error
}
