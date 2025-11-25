package avro

import (
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"go.uber.org/fx"
)

// Module provides Avro deserialization components for dependency injection
var Module = fx.Module("avro",
	fx.Provide(
		encoding.NewConfluentWireFormatParser,
		encoding.NewConfluentWireFormatBuilder,
		encoding.NewHambaDecoder,
		encoding.NewHambaEncoder,
		deserialization.NewRegistrySchemaResolver,
		serialization.NewTypeSchemaRegistry,
		deserialization.NewDeserializer,
		serialization.NewAvroSerializer,
	),
)
