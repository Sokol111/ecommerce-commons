package avro

import (
	"context"
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/deserialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro/serialization"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/events"
	"github.com/twmb/franz-go/pkg/sr"
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

func provideSchemaRegistryClient(kafkaConf config.Config, log *zap.Logger) (*sr.Client, error) {
	client, err := sr.NewClient(sr.URLs(kafkaConf.SchemaRegistry.URL))
	if err != nil {
		return nil, fmt.Errorf("failed to create schema registry client: %w", err)
	}
	return client, nil
}

func provideSchemaRegistry(lc fx.Lifecycle, client *sr.Client, log *zap.Logger, cm health.ComponentManager, eventRegistry events.EventRegistry) serialization.SchemaRegistry {
	registry := serialization.NewSchemaRegistry(client)

	markReady := cm.AddComponent("schema_registry")
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
