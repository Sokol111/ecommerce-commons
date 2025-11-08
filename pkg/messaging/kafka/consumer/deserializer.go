package consumer

import (
	"fmt"
	"reflect"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/config"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

// Deserializer deserializes Avro bytes to Go structs using Schema Registry
type Deserializer interface {
	// Deserialize deserializes Avro bytes to a Go struct
	//
	// The subject parameter is used to fetch the schema from Schema Registry.
	// The data must be in format: [0x00][schema_id (4 bytes)][avro_data]
	//
	// Returns a concrete Go type based on the type registry configuration.
	Deserialize(subject string, data []byte) (interface{}, error)

	// Close releases resources used by the deserializer
	Close() error
}

// TypeMapping maps Avro schema full names to Go types
type TypeMapping map[string]reflect.Type

type avroDeserializer struct {
	deserializer *avro.GenericDeserializer
}

// NewAvroDeserializer creates a new Avro deserializer with Schema Registry integration
//
// Example:
//
//	deserializer, err := NewAvroDeserializer(config, consumer.TypeMapping{
//	   "com.ecommerce.events.product.ProductCreatedEvent": reflect.TypeOf(events.ProductCreatedEvent{}),
//	   "com.ecommerce.events.product.ProductUpdatedEvent": reflect.TypeOf(events.ProductUpdatedEvent{}),
//	})
func NewAvroDeserializer(conf config.SchemaRegistryConfig, typeMap TypeMapping) (Deserializer, error) {
	client, err := schemaregistry.NewClient(schemaregistry.NewConfig(conf.URL))
	if err != nil {
		return nil, fmt.Errorf("failed to create schema registry client: %w", err)
	}

	deserConfig := avro.NewDeserializerConfig()

	deser, err := avro.NewGenericDeserializer(client, serde.ValueSerde, deserConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create avro deserializer: %w", err)
	}

	return &avroDeserializer{deserializer: deser}, nil
}

func (d *avroDeserializer) Deserialize(subject string, data []byte) (interface{}, error) {
	return d.deserializer.Deserialize(subject, data)
}

func (d *avroDeserializer) Close() error {
	if d.deserializer != nil {
		d.deserializer.Close()
	}
	return nil
}
