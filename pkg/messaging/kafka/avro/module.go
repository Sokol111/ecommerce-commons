package avro

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/encoding"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/mapping"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewAvroModule provides Avro serialization and deserialization components for dependency injection.
func NewAvroModule() fx.Option {
	return fx.Module("avro",
		fx.Provide(
			provideSchemaRegistryClient,
			provideConfluentRegistry,
			mapping.NewTypeMapping,
			encoding.NewConfluentWireFormat,
			encoding.NewHambaDecoder,
			encoding.NewHambaEncoder,
			deserialization.NewWriterSchemaResolver,
			deserialization.NewDeserializer,
			serialization.NewSerializer,
		),
	)
}

func provideSchemaRegistryClient(lc fx.Lifecycle, kafkaConf config.Config, log *zap.Logger) (schemaregistry.Client, error) {
	config := schemaregistry.NewConfig(kafkaConf.SchemaRegistry.URL)
	config.RequestTimeoutMs = 5000 // 5 seconds timeout for Schema Registry requests

	client, err := schemaregistry.NewClient(config)
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

func provideConfluentRegistry(lc fx.Lifecycle, client schemaregistry.Client, typeMapping *mapping.TypeMapping, kafkaConf config.Config, log *zap.Logger, cm health.ComponentManager) (serialization.ConfluentRegistry, error) {
	var registry = serialization.NewConfluentRegistry(client, typeMapping)

	markReady := cm.AddComponent("confluent_schema_registry")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if kafkaConf.SchemaRegistry.AutoRegisterSchemas {
				err := registry.RegisterAllSchemasAtStartup()
				if err != nil {
					return err
				}
				log.Info("all schemas registered at startup")
			} else {
				log.Info("auto-registration of schemas is disabled")
			}
			markReady()
			return nil
		},
	})

	return registry, nil
}
