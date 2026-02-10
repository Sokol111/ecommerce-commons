package avro

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	avroserialization "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewAvroModule provides Avro serialization and deserialization components for dependency injection.
// Events should be registered via event API modules (e.g., catalog_events.Module()).
func NewAvroModule() fx.Option {
	return fx.Module("avro",
		fx.Provide(
			provideSchemaRegistryClient,
			provideSchemaRegistry,
			events.NewEventRegistry,
			deserialization.NewWriterSchemaResolver,
			deserialization.NewDeserializer,
			provideSerializer,
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

func provideSchemaRegistry(lc fx.Lifecycle, client schemaregistry.Client, log *zap.Logger, cm health.ComponentManager) avroserialization.SchemaRegistry {
	registry := avroserialization.NewConfluentRegistry(client)

	markReady := cm.AddComponent("confluent_schema_registry")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Schemas are now registered on-demand during serialization
			// or via event API modules at startup
			log.Info("schema registry client ready")
			markReady()
			return nil
		},
	})

	return registry
}

func provideSerializer(schemaRegistry avroserialization.SchemaRegistry) serde.Serializer {
	return avroserialization.NewSerializer(schemaRegistry)
}
