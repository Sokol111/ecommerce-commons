package kafkaproto

import (
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	"go.uber.org/fx"
)

// NewProtoModule provides proto-based Serializer and Deserializer for dependency injection.
func NewProtoModule() fx.Option {
	return fx.Module("kafkaproto",
		fx.Provide(
			func() serde.Serializer { return NewSerializer() },
			func() serde.Deserializer { return NewDeserializer() },
		),
	)
}
