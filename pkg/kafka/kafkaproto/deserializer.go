package kafkaproto

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type protoDeserializer struct{}

// NewDeserializer creates a Deserializer that uses protoregistry.GlobalTypes and proto.Unmarshal.
// The "event_type" header must contain the proto full name (e.g. "tenant.v1.TenantUpdatedEvent").
func NewDeserializer() *protoDeserializer {
	return &protoDeserializer{}
}

func (d *protoDeserializer) Deserialize(data []byte, headers map[string][]byte) (proto.Message, error) {
	eventTypeBytes, ok := headers["event_type"]
	if !ok {
		return nil, fmt.Errorf("missing required header: event_type")
	}

	msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(eventTypeBytes))
	if err != nil {
		return nil, fmt.Errorf("unknown event type %q: %w", eventTypeBytes, err)
	}

	msg := msgType.New().Interface()
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("proto unmarshal failed for %q: %w", eventTypeBytes, err)
	}

	return msg, nil
}
