package avro

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides Avro deserialization components for dependency injection
func NewAvroModule() fx.Option {
	return fx.Module("avro",
		fx.Provide(
			provideSchemaRegistryClient,
			encoding.NewConfluentWireFormat,
			encoding.NewHambaDecoder,
			encoding.NewHambaEncoder,
			deserialization.NewRegistrySchemaResolver,
			serialization.NewTypeSchemaRegistry,
			deserialization.NewDeserializer,
			serialization.NewAvroSerializer,
		),
	)
}

func provideSchemaRegistryClient(lc fx.Lifecycle, kafkaConf config.Config, log *zap.Logger) (schemaregistry.Client, error) {
	client, err := schemaregistry.NewClient(schemaregistry.NewConfig(kafkaConf.SchemaRegistry.URL))
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info("closing schema registry client")
			return client.Close()
		},
	})

	return client, nil
}
