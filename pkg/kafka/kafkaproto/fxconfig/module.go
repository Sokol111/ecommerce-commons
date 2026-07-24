package fxconfig

import (
	"github.com/Sokol111/ecommerce-commons/pkg/kafka/kafkaproto"
	"go.uber.org/fx"
)

// NewProtoModule provides proto-based Serializer and Deserializer for dependency injection.
func NewProtoModule() fx.Option {
	return fx.Module("kafkaproto",
		fx.Provide(
			func() kafkaproto.Serializer { return kafkaproto.NewSerializer() },
			func() kafkaproto.Deserializer { return kafkaproto.NewDeserializer() },
		),
	)
}
