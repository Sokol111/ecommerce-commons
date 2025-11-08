package consumer

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// UnmarshalFunc is a function that deserializes bytes using a schema name.
// It matches the signature of events.Unmarshal from the generated API package.
type UnmarshalFunc func(schemaName string, data []byte, v interface{}) error

// NewDeserializer creates a Deserializer that uses the provided unmarshal function.
// The event-type header is passed as schemaName to the unmarshal function.
// Returns ErrSkipMessage if event-type header is missing or empty.
//
// Example usage:
//
//	deserializer := consumer.NewDeserializer(events.Unmarshal)
func NewDeserializer(unmarshal UnmarshalFunc) Deserializer {
	return func(data []byte, headers []kafka.Header) (any, error) {
		eventType := GetEventType(headers)
		if eventType == "" {
			return nil, fmt.Errorf("missing event-type header: %w", ErrSkipMessage)
		}

		var event any
		if err := unmarshal(eventType, data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event of type %s: %w: %w", eventType, err, ErrSkipMessage)
		}

		return event, nil
	}
}
