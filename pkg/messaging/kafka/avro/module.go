package avro

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
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

func provideSchemaRegistry(lc fx.Lifecycle, client schemaregistry.Client, log *zap.Logger, cm health.ComponentManager, eventRegistry events.EventRegistry) avroserialization.SchemaRegistry {
	registry := serialization.NewConfluentRegistry(client)

	markReady := cm.AddComponent("confluent_schema_registry")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Register all event schemas on startup
			for _, event := range eventRegistry.All() {
				schemaName := event.GetSchemaName()
				schemaJSON := event.GetSchema()

				if _, err := registry.GetOrRegisterSchema(schemaName, schemaJSON); err != nil {
					return fmt.Errorf("failed to register schema %s: %w", schemaName, err)
				}
				log.Info("registered schema", zap.String("schema", schemaName))
			}

			markReady()
			return nil
		},
	})

	return registry
}
