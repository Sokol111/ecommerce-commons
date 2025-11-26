package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyDefaults_SchemaRegistry(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
	}

	applyDefaults(cfg)

	assert.Equal(t, defaultSchemaRegistryCacheCapacity, cfg.SchemaRegistry.CacheCapacity)
}

func TestApplyDefaults_SchemaRegistryCustomValue(t *testing.T) {
	customCapacity := 5000
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL:           "http://localhost:8081",
			CacheCapacity: customCapacity,
		},
	}

	applyDefaults(cfg)

	assert.Equal(t, customCapacity, cfg.SchemaRegistry.CacheCapacity, "should not override custom value")
}

func TestApplyDefaults_GlobalConsumerConfig(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ConsumersConfig: ConsumersConfig{},
	}

	applyDefaults(cfg)

	assert.Equal(t, defaultMaxRetryAttempts, cfg.ConsumersConfig.DefaultMaxRetryAttempts)
	assert.Equal(t, defaultInitialBackoff, cfg.ConsumersConfig.DefaultInitialBackoff)
	assert.Equal(t, defaultMaxBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	assert.Equal(t, defaultChannelBufferSize, cfg.ConsumersConfig.DefaultChannelBufferSize)
}

func TestApplyDefaults_GlobalConsumerConfigCustomValues(t *testing.T) {
	customRetries := 10
	customInitialBackoff := 5 * time.Second
	customMaxBackoff := 2 * time.Minute
	customBufferSize := 500

	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ConsumersConfig: ConsumersConfig{
			DefaultMaxRetryAttempts:  customRetries,
			DefaultInitialBackoff:    customInitialBackoff,
			DefaultMaxBackoff:        customMaxBackoff,
			DefaultChannelBufferSize: customBufferSize,
		},
	}

	applyDefaults(cfg)

	assert.Equal(t, customRetries, cfg.ConsumersConfig.DefaultMaxRetryAttempts)
	assert.Equal(t, customInitialBackoff, cfg.ConsumersConfig.DefaultInitialBackoff)
	assert.Equal(t, customMaxBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	assert.Equal(t, customBufferSize, cfg.ConsumersConfig.DefaultChannelBufferSize)
}

func TestApplyDefaults_ProducerConfig(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ProducerConfig: ProducerConfig{},
	}

	applyDefaults(cfg)

	assert.Equal(t, defaultProducerReadinessTimeout, cfg.ProducerConfig.ReadinessTimeoutSeconds)
}

func TestApplyDefaults_ProducerConfigCustomValue(t *testing.T) {
	customTimeout := 120
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ProducerConfig: ProducerConfig{
			ReadinessTimeoutSeconds: customTimeout,
		},
	}

	applyDefaults(cfg)

	assert.Equal(t, customTimeout, cfg.ProducerConfig.ReadinessTimeoutSeconds)
}

func TestApplyConsumerDefaults_AllDefaults(t *testing.T) {
	globalConfig := &ConsumersConfig{
		DefaultGroupID:           "default-group",
		DefaultAutoOffsetReset:   "earliest",
		DefaultMaxRetryAttempts:  5,
		DefaultInitialBackoff:    2 * time.Second,
		DefaultMaxBackoff:        1 * time.Minute,
		DefaultChannelBufferSize: 200,
	}

	consumer := &ConsumerConfig{
		Name:  "test-consumer",
		Topic: "test-topic",
	}

	applyConsumerDefaults(consumer, globalConfig)

	assert.Equal(t, "default-group", consumer.GroupID)
	assert.Equal(t, "earliest", consumer.AutoOffsetReset)
	assert.Equal(t, defaultConsumerReadinessTimeout, consumer.ReadinessTimeoutSeconds)
	assert.Equal(t, 5, consumer.MaxRetryAttempts)
	assert.Equal(t, 2*time.Second, consumer.InitialBackoff)
	assert.Equal(t, 1*time.Minute, consumer.MaxBackoff)
	assert.Equal(t, 200, consumer.ChannelBufferSize)
}

func TestApplyConsumerDefaults_CustomValues(t *testing.T) {
	globalConfig := &ConsumersConfig{
		DefaultGroupID:           "default-group",
		DefaultAutoOffsetReset:   "earliest",
		DefaultMaxRetryAttempts:  5,
		DefaultInitialBackoff:    2 * time.Second,
		DefaultMaxBackoff:        1 * time.Minute,
		DefaultChannelBufferSize: 200,
	}

	consumer := &ConsumerConfig{
		Name:                    "test-consumer",
		Topic:                   "test-topic",
		GroupID:                 "custom-group",
		AutoOffsetReset:         "latest",
		ReadinessTimeoutSeconds: 120,
		MaxRetryAttempts:        10,
		InitialBackoff:          5 * time.Second,
		MaxBackoff:              2 * time.Minute,
		ChannelBufferSize:       500,
	}

	applyConsumerDefaults(consumer, globalConfig)

	// Should keep custom values
	assert.Equal(t, "custom-group", consumer.GroupID)
	assert.Equal(t, "latest", consumer.AutoOffsetReset)
	assert.Equal(t, 120, consumer.ReadinessTimeoutSeconds)
	assert.Equal(t, 10, consumer.MaxRetryAttempts)
	assert.Equal(t, 5*time.Second, consumer.InitialBackoff)
	assert.Equal(t, 2*time.Minute, consumer.MaxBackoff)
	assert.Equal(t, 500, consumer.ChannelBufferSize)
}

func TestApplyConsumerDefaults_DLQTopicNaming(t *testing.T) {
	globalConfig := &ConsumersConfig{}

	tests := []struct {
		name             string
		consumer         ConsumerConfig
		expectedDLQTopic string
	}{
		{
			name: "DLQ enabled, no custom topic",
			consumer: ConsumerConfig{
				Name:      "consumer1",
				Topic:     "orders",
				EnableDLQ: true,
			},
			expectedDLQTopic: "orders.dlq",
		},
		{
			name: "DLQ enabled, custom topic",
			consumer: ConsumerConfig{
				Name:      "consumer2",
				Topic:     "payments",
				EnableDLQ: true,
				DLQTopic:  "custom.dlq",
			},
			expectedDLQTopic: "custom.dlq",
		},
		{
			name: "DLQ disabled",
			consumer: ConsumerConfig{
				Name:      "consumer3",
				Topic:     "notifications",
				EnableDLQ: false,
			},
			expectedDLQTopic: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := tt.consumer
			applyConsumerDefaults(&consumer, globalConfig)
			assert.Equal(t, tt.expectedDLQTopic, consumer.DLQTopic)
		})
	}
}

func TestApplyDefaults_MultipleConsumers(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ConsumersConfig: ConsumersConfig{
			DefaultGroupID:           "default-group",
			DefaultAutoOffsetReset:   "earliest",
			DefaultMaxRetryAttempts:  3,
			DefaultInitialBackoff:    1 * time.Second,
			DefaultMaxBackoff:        30 * time.Second,
			DefaultChannelBufferSize: 100,
			ConsumerConfig: []ConsumerConfig{
				{
					Name:  "consumer1",
					Topic: "topic1",
				},
				{
					Name:                    "consumer2",
					Topic:                   "topic2",
					GroupID:                 "custom-group",
					MaxRetryAttempts:        5,
					ReadinessTimeoutSeconds: 120,
				},
				{
					Name:      "consumer3",
					Topic:     "topic3",
					EnableDLQ: true,
				},
			},
		},
	}

	applyDefaults(cfg)

	// Consumer 1 - all defaults
	assert.Equal(t, "default-group", cfg.ConsumersConfig.ConsumerConfig[0].GroupID)
	assert.Equal(t, "earliest", cfg.ConsumersConfig.ConsumerConfig[0].AutoOffsetReset)
	assert.Equal(t, 3, cfg.ConsumersConfig.ConsumerConfig[0].MaxRetryAttempts)

	// Consumer 2 - some custom values
	assert.Equal(t, "custom-group", cfg.ConsumersConfig.ConsumerConfig[1].GroupID)
	assert.Equal(t, 5, cfg.ConsumersConfig.ConsumerConfig[1].MaxRetryAttempts)
	assert.Equal(t, 120, cfg.ConsumersConfig.ConsumerConfig[1].ReadinessTimeoutSeconds)
	assert.Equal(t, 1*time.Second, cfg.ConsumersConfig.ConsumerConfig[1].InitialBackoff) // From global

	// Consumer 3 - DLQ enabled
	assert.Equal(t, "topic3.dlq", cfg.ConsumersConfig.ConsumerConfig[2].DLQTopic)
}

func TestApplyDefaults_CompleteConfig(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL: "http://localhost:8081",
		},
		ConsumersConfig: ConsumersConfig{
			ConsumerConfig: []ConsumerConfig{
				{Name: "consumer1", Topic: "topic1"},
			},
		},
		ProducerConfig: ProducerConfig{},
	}

	applyDefaults(cfg)

	// Verify all defaults are applied
	assert.Equal(t, defaultSchemaRegistryCacheCapacity, cfg.SchemaRegistry.CacheCapacity)
	assert.Equal(t, defaultMaxRetryAttempts, cfg.ConsumersConfig.DefaultMaxRetryAttempts)
	assert.Equal(t, defaultInitialBackoff, cfg.ConsumersConfig.DefaultInitialBackoff)
	assert.Equal(t, defaultMaxBackoff, cfg.ConsumersConfig.DefaultMaxBackoff)
	assert.Equal(t, defaultChannelBufferSize, cfg.ConsumersConfig.DefaultChannelBufferSize)
	assert.Equal(t, defaultProducerReadinessTimeout, cfg.ProducerConfig.ReadinessTimeoutSeconds)

	// Verify consumer inherits from global defaults
	assert.Equal(t, defaultMaxRetryAttempts, cfg.ConsumersConfig.ConsumerConfig[0].MaxRetryAttempts)
	assert.Equal(t, defaultInitialBackoff, cfg.ConsumersConfig.ConsumerConfig[0].InitialBackoff)
	assert.Equal(t, defaultMaxBackoff, cfg.ConsumersConfig.ConsumerConfig[0].MaxBackoff)
	assert.Equal(t, defaultChannelBufferSize, cfg.ConsumersConfig.ConsumerConfig[0].ChannelBufferSize)
	assert.Equal(t, defaultConsumerReadinessTimeout, cfg.ConsumersConfig.ConsumerConfig[0].ReadinessTimeoutSeconds)
}
