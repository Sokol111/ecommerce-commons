package kafkaproto

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

type protoSerializer struct{}

// NewSerializer creates a Serializer that uses proto.Marshal.
func NewSerializer() *protoSerializer {
	return &protoSerializer{}
}

func (s *protoSerializer) Serialize(msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("proto marshal failed: %w", err)
	}
	return data, nil
}
