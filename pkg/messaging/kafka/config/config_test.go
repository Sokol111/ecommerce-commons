package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewConfig_ValidYAML(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092,localhost:9093"
  schema-registry:
    url: "http://schema-registry:8081"
    cache-capacity: 2000
    auto_register_schemas: true
  consumers-config:
    default-group-id: "test-group"
    default-auto-offset-reset: "earliest"
    default-max-retries: 5
    default-initial-backoff: 2s
    default-max-backoff: 1m
    default-channel-buffer-size: 200
    consumers:
      - name: "product-consumer"
        topic: "catalog.product.events"
        group-id: "product-service"
        auto-offset-reset: "latest"
        enable-dlq: true
        dlq-topic: "catalog.product.events.dlq"
        readiness-timeout-seconds: 120
        max-retries: 10
        initial-backoff: 500ms
        max-backoff: 2m
        channel-buffer-size: 500
  producer-config:
    readiness-timeout-seconds: 60
    fail-on-broker-error: true
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg, err := newConfig(v, logger)

	require.NoError(t, err)
	assert.Equal(t, "localhost:9092,localhost:9093", cfg.Brokers)
	assert.Equal(t, "http://schema-registry:8081", cfg.SchemaRegistry.URL)
	assert.Equal(t, 2000, cfg.SchemaRegistry.CacheCapacity)
	assert.True(t, cfg.SchemaRegistry.AutoRegisterSchemas)

	// Global consumer config
	assert.Equal(t, "test-group", cfg.ConsumersConfig.DefaultGroupID)
	assert.Equal(t, "earliest", cfg.ConsumersConfig.DefaultAutoOffsetReset)
	assert.Equal(t, uint(5), *cfg.ConsumersConfig.DefaultMaxRetries)
	assert.Equal(t, 2*time.Second, cfg.ConsumersConfig.DefaultInitialBackoff)
	assert.Equal(t, 1*time.Minute, cfg.ConsumersConfig.DefaultMaxBackoff)
	assert.Equal(t, 200, cfg.ConsumersConfig.DefaultChannelBufferSize)

	// Individual consumer
	require.Len(t, cfg.ConsumersConfig.ConsumerConfig, 1)
	consumer := cfg.ConsumersConfig.ConsumerConfig[0]
	assert.Equal(t, "product-consumer", consumer.Name)
	assert.Equal(t, "catalog.product.events", consumer.Topic)
	assert.Equal(t, "product-service", consumer.GroupID)
	assert.Equal(t, "latest", consumer.AutoOffsetReset)
	assert.True(t, consumer.EnableDLQ)
	assert.Equal(t, "catalog.product.events.dlq", consumer.DLQTopic)
	assert.Equal(t, 120, consumer.ReadinessTimeoutSeconds)
	assert.Equal(t, uint(10), *consumer.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, consumer.InitialBackoff)
	assert.Equal(t, 2*time.Minute, consumer.MaxBackoff)
	assert.Equal(t, 500, consumer.ChannelBufferSize)

	// Producer config
	assert.Equal(t, 60, cfg.ProducerConfig.ReadinessTimeoutSeconds)
	assert.True(t, cfg.ProducerConfig.FailOnBrokerError)
}

func TestNewConfig_MinimalYAML(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg, err := newConfig(v, logger)

	require.NoError(t, err)

	// Required fields
	assert.Equal(t, "localhost:9092", cfg.Brokers)
	assert.Equal(t, "http://schema-registry:8081", cfg.SchemaRegistry.URL)

	// Defaults should be applied
	assert.Equal(t, defaultSchemaRegistryCacheCapacity, cfg.SchemaRegistry.CacheCapacity)
	assert.Equal(t, defaultMaxRetries, *cfg.ConsumersConfig.DefaultMaxRetries)
	assert.Equal(t, defaultInitialBackoff, cfg.ConsumersConfig.DefaultInitialBackoff)
	assert.Equal(t, defaultMaxBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	assert.Equal(t, defaultChannelBufferSize, cfg.ConsumersConfig.DefaultChannelBufferSize)
	assert.Equal(t, defaultProducerReadinessTimeout, cfg.ProducerConfig.ReadinessTimeoutSeconds)
}

func TestNewConfig_InvalidYAML_MissingBrokers(t *testing.T) {
	yamlConfig := `
kafka:
  schema-registry:
    url: "http://schema-registry:8081"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "brokers cannot be empty")
}

func TestNewConfig_InvalidYAML_MissingSchemaRegistryURL(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    cache-capacity: 1000
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema registry URL cannot be empty")
}

func TestNewConfig_ConsumerWithDefaults(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
  consumers-config:
    default-group-id: "default-group"
    default-auto-offset-reset: "earliest"
    consumers:
      - name: "test-consumer"
        topic: "test-topic"
        enable-dlq: true
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg, err := newConfig(v, logger)

	require.NoError(t, err)
	require.Len(t, cfg.ConsumersConfig.ConsumerConfig, 1)

	consumer := cfg.ConsumersConfig.ConsumerConfig[0]
	// Should inherit from global defaults
	assert.Equal(t, "default-group", consumer.GroupID)
	assert.Equal(t, "earliest", consumer.AutoOffsetReset)
	assert.Equal(t, defaultMaxRetries, *consumer.MaxRetries)
	assert.Equal(t, defaultInitialBackoff, consumer.InitialBackoff)
	assert.Equal(t, defaultMaxBackoff, consumer.MaxBackoff)
	assert.Equal(t, defaultChannelBufferSize, consumer.ChannelBufferSize)
	assert.Equal(t, defaultConsumerReadinessTimeout, consumer.ReadinessTimeoutSeconds)

	// DLQ topic should be auto-generated
	assert.Equal(t, "test-topic.dlq", consumer.DLQTopic)
}

func TestNewConfig_InvalidConsumerConfig(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
  consumers-config:
    consumers:
      - name: ""
        topic: "test-topic"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestNewConfig_InvalidSchemaCapacity(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
    cache-capacity: 50
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache capacity must be between")
}

func TestNewConfig_InvalidBackoffRelationship(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
  consumers-config:
    default-initial-backoff: 10s
    default-max-backoff: 5s
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "initial backoff")
	assert.Contains(t, err.Error(), "cannot be greater than max backoff")
}

func TestNewConfig_MultipleConsumers(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
  consumers-config:
    default-group-id: "default-group"
    consumers:
      - name: "consumer1"
        topic: "topic1"
      - name: "consumer2"
        topic: "topic2"
        group-id: "custom-group"
      - name: "consumer3"
        topic: "topic3"
        enable-dlq: true
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg, err := newConfig(v, logger)

	require.NoError(t, err)
	require.Len(t, cfg.ConsumersConfig.ConsumerConfig, 3)

	// Consumer 1 - uses defaults
	assert.Equal(t, "default-group", cfg.ConsumersConfig.ConsumerConfig[0].GroupID)

	// Consumer 2 - custom group
	assert.Equal(t, "custom-group", cfg.ConsumersConfig.ConsumerConfig[1].GroupID)

	// Consumer 3 - DLQ enabled
	assert.True(t, cfg.ConsumersConfig.ConsumerConfig[2].EnableDLQ)
	assert.Equal(t, "topic3.dlq", cfg.ConsumersConfig.ConsumerConfig[2].DLQTopic)
}

func TestNewConfig_InvalidAutoOffsetReset(t *testing.T) {
	yamlConfig := `
kafka:
  brokers: "localhost:9092"
  schema-registry:
    url: "http://schema-registry:8081"
  consumers-config:
    consumers:
      - name: "consumer1"
        topic: "topic1"
        auto-offset-reset: "invalid"
`

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBufferString(yamlConfig))
	require.NoError(t, err)

	logger := zap.NewNop()
	_, err = newConfig(v, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auto offset reset must be 'earliest' or 'latest'")
}

func TestNewKafkaConfigModule(t *testing.T) {
	module := NewKafkaConfigModule()
	assert.NotNil(t, module)
}
