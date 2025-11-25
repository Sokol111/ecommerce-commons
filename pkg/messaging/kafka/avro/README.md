# Avro Package

Пакет `avro` надає компоненти для серіалізації та десеріалізації Avro повідомлень з використанням Confluent Schema Registry.

## Компоненти

### Deserializer
Головний інтерфейс для десеріалізації Avro повідомлень у Go структури.

```go
type Deserializer interface {
    Deserialize(data []byte) (interface{}, error)
}
```

### Decoder
Декодує Avro байти у Go об'єкти на основі схеми.

```go
type Decoder interface {
    Decode(payload []byte, metadata *SchemaMetadata) (interface{}, error)
}
```

### SchemaResolver
Резолвить schema ID у метадані схеми з Schema Registry.

```go
type SchemaResolver interface {
    Resolve(schemaID int) (*SchemaMetadata, error)
}
```

### WireFormatParser
Парсить Confluent wire format для витягування schema ID та payload.

```go
type WireFormatParser interface {
    Parse(data []byte) (schemaID int, payload []byte, err error)
}
```

## Використання

### Базове використання з fx DI

```go
import (
    "github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/avro"
    "go.uber.org/fx"
)

// Реєструємо type mapping
typeMapping := avro.TypeMapping{
    "com.ecommerce.events.ProductCreatedEvent": reflect.TypeOf(ProductCreatedEvent{}),
    "com.ecommerce.events.ProductUpdatedEvent": reflect.TypeOf(ProductUpdatedEvent{}),
}

app := fx.New(
    fx.Provide(
        // Надаємо Schema Registry клієнт
        func() (schemaregistry.Client, error) {
            return avro.ProvideSchemaRegistryClient("http://localhost:8081")
        },
        // Надаємо type mapping
        func() avro.TypeMapping {
            return typeMapping
        },
    ),
    // Включаємо avro модуль
    avro.Module,
)
```

### Використання напряму

```go
// Створюємо Schema Registry клієнт
client, _ := schemaregistry.NewClient(schemaregistry.NewConfig("http://localhost:8081"))

// Створюємо type mapping
typeMapping := avro.TypeMapping{
    "com.ecommerce.events.ProductCreatedEvent": reflect.TypeOf(ProductCreatedEvent{}),
}

// Створюємо resolver та deserializer
resolver := avro.NewRegistrySchemaResolver(client, typeMapping)
deserializer := avro.NewDeserializer(resolver)

// Десеріалізуємо повідомлення
event, err := deserializer.Deserialize(kafkaMessage.Value)
if err != nil {
    log.Fatal(err)
}
```

## Wire Format

Пакет підтримує Confluent wire format:
```
[0x00][schema_id (4 bytes)][avro_data]
```

- Byte 0: Magic byte (завжди 0x00)
- Bytes 1-4: Schema ID (big-endian)
- Bytes 5+: Avro закодовані дані

## Кешування

`SchemaResolver` автоматично кешує схеми для оптимізації продуктивності. Немає необхідності в додатковому кешуванні на рівні додатку.

## Залежності

- `github.com/hamba/avro/v2` - Avro кодування/декодування
- `github.com/confluentinc/confluent-kafka-go/v2/schemaregistry` - Schema Registry клієнт
