package avro

import (
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"go.uber.org/fx"
)

// Module provides Avro deserialization components for dependency injection
var Module = fx.Module("avro",
	fx.Provide(
		NewRegistrySchemaResolver,
		NewDeserializer,
		fx.Private,
	),
)

// ProvideSchemaRegistryClient creates a Schema Registry client
// This is a helper function that can be used in consumer modules
func ProvideSchemaRegistryClient(url string) (schemaregistry.Client, error) {
	return schemaregistry.NewClient(schemaregistry.NewConfig(url))
}
