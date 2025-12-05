package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig_Success(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092",
		SchemaRegistry: SchemaRegistryConfig{
			URL:           "http://schema-registry:8081",
			CacheCapacity: 1000,
		},
		ConsumersConfig: ConsumersConfig{
			DefaultGroupID:           "test-group",
			DefaultAutoOffsetReset:   "earliest",
			DefaultMaxRetries:        uintPtr(3),
			DefaultInitialBackoff:    1 * time.Second,
			DefaultMaxBackoff:        30 * time.Second,
			DefaultChannelBufferSize: 100,
		},
		ProducerConfig: ProducerConfig{
			ReadinessTimeoutSeconds: 30,
		},
	}

	err := validateConfig(cfg)
	assert.NoError(t, err)
}

func TestValidateBrokers_EmptyBrokers(t *testing.T) {
	tests := []struct {
		name    string
		brokers string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs and spaces", "\t  \n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Brokers: tt.brokers}
			err := validateBrokers(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "kafka brokers cannot be empty")
		})
	}
}

func TestValidateBrokers_Success(t *testing.T) {
	cfg := &Config{Brokers: "localhost:9092"}
	err := validateBrokers(cfg)
	assert.NoError(t, err)
}

func TestValidateSchemaRegistry_EmptyURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SchemaRegistryConfig{
				URL:           tt.url,
				CacheCapacity: 1000,
			}
			err := validateSchemaRegistry(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "schema registry URL cannot be empty")
		})
	}
}

func TestValidateSchemaRegistry_InvalidCacheCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"below minimum", 50},
		{"at minimum - 1", minSchemaCacheCapacity - 1},
		{"above maximum", maxSchemaCacheCapacity + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SchemaRegistryConfig{
				URL:           "http://localhost:8081",
				CacheCapacity: tt.capacity,
			}
			err := validateSchemaRegistry(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cache capacity must be between")
		})
	}
}

func TestValidateSchemaRegistry_ValidCacheCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"minimum", minSchemaCacheCapacity},
		{"middle", 5000},
		{"maximum", maxSchemaCacheCapacity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SchemaRegistryConfig{
				URL:           "http://localhost:8081",
				CacheCapacity: tt.capacity,
			}
			err := validateSchemaRegistry(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidateGlobalConsumerConfig_MaxRetries(t *testing.T) {
	tests := []struct {
		name          string
		retryAttempts *uint
		expectError   bool
	}{
		{"nil (valid)", nil, false},
		{"zero (valid)", uintPtr(0), false},
		{"middle", uintPtr(50), false},
		{"at maximum", uintPtr(maxMaxRetries), false},
		{"above maximum", uintPtr(maxMaxRetries + 1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConsumersConfig{
				DefaultMaxRetries: tt.retryAttempts,
			}
			err := validateGlobalConsumerConfig(cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "retries must be between")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGlobalConsumerConfig_InitialBackoff(t *testing.T) {
	tests := []struct {
		name        string
		backoff     time.Duration
		expectError bool
	}{
		{"zero (will use default)", 0, false},
		{"below minimum", 50 * time.Millisecond, true},
		{"at minimum", minInitialBackoff, false},
		{"middle", 5 * time.Second, false},
		{"at maximum", maxInitialBackoff, false},
		{"above maximum", 31 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConsumersConfig{
				DefaultInitialBackoff: tt.backoff,
			}
			err := validateGlobalConsumerConfig(cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "initial backoff must be between")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGlobalConsumerConfig_MaxBackoff(t *testing.T) {
	tests := []struct {
		name        string
		backoff     time.Duration
		expectError bool
	}{
		{"zero (will use default)", 0, false},
		{"below minimum", 500 * time.Millisecond, true},
		{"at minimum", minMaxBackoff, false},
		{"middle", 2 * time.Minute, false},
		{"at maximum", maxMaxBackoffDuration, false},
		{"above maximum", 6 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConsumersConfig{
				DefaultMaxBackoff: tt.backoff,
			}
			err := validateGlobalConsumerConfig(cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "max backoff must be between")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGlobalConsumerConfig_BackoffRelationship(t *testing.T) {
	cfg := &ConsumersConfig{
		DefaultInitialBackoff: 10 * time.Second,
		DefaultMaxBackoff:     5 * time.Second,
	}
	err := validateGlobalConsumerConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial backoff")
	assert.Contains(t, err.Error(), "cannot be greater than max backoff")
}

func TestValidateGlobalConsumerConfig_ChannelBufferSize(t *testing.T) {
	tests := []struct {
		name        string
		bufferSize  int
		expectError bool
	}{
		{"zero (will use default)", 0, false},
		{"below minimum", minChannelBufferSize - 1, true},
		{"at minimum", minChannelBufferSize, false},
		{"middle", 500, false},
		{"at maximum", maxChannelBufferSize, false},
		{"above maximum", maxChannelBufferSize + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ConsumersConfig{
				DefaultChannelBufferSize: tt.bufferSize,
			}
			err := validateGlobalConsumerConfig(cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "channel buffer size must be between")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConsumer_EmptyName(t *testing.T) {
	consumer := &ConsumerConfig{
		Name:  "",
		Topic: "test-topic",
	}
	err := validateConsumer(0, consumer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestValidateConsumer_EmptyTopic(t *testing.T) {
	consumer := &ConsumerConfig{
		Name:  "test-consumer",
		Topic: "",
	}
	err := validateConsumer(0, consumer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic cannot be empty")
}

func TestValidateConsumer_InvalidAutoOffsetReset(t *testing.T) {
	tests := []string{"beginning", "end", "invalid", "first"}

	for _, offset := range tests {
		t.Run(offset, func(t *testing.T) {
			consumer := &ConsumerConfig{
				Name:            "test-consumer",
				Topic:           "test-topic",
				AutoOffsetReset: offset,
			}
			err := validateConsumer(0, consumer)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "auto offset reset must be 'earliest' or 'latest'")
		})
	}
}

func TestValidateConsumer_ValidAutoOffsetReset(t *testing.T) {
	tests := []string{"earliest", "latest", ""}

	for _, offset := range tests {
		t.Run(offset, func(t *testing.T) {
			consumer := &ConsumerConfig{
				Name:            "test-consumer",
				Topic:           "test-topic",
				AutoOffsetReset: offset,
			}
			err := validateConsumer(0, consumer)
			assert.NoError(t, err)
		})
	}
}

func TestValidateConsumer_ReadinessTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		expectError bool
	}{
		{"zero (no timeout)", 0, false},
		{"valid timeout", 60, false},
		{"maximum timeout", maxReadinessTimeout, false},
		{"exceeds maximum", maxReadinessTimeout + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &ConsumerConfig{
				Name:                    "test-consumer",
				Topic:                   "test-topic",
				ReadinessTimeoutSeconds: tt.timeout,
			}
			err := validateConsumer(0, consumer)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "readiness timeout cannot exceed")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConsumer_DLQTopicSameAsMainTopic(t *testing.T) {
	consumer := &ConsumerConfig{
		Name:      "test-consumer",
		Topic:     "test-topic",
		EnableDLQ: true,
		DLQTopic:  "test-topic",
	}
	err := validateConsumer(0, consumer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ topic cannot be the same as main topic")
}

func TestValidateConsumer_DLQTopicDifferent(t *testing.T) {
	consumer := &ConsumerConfig{
		Name:      "test-consumer",
		Topic:     "test-topic",
		EnableDLQ: true,
		DLQTopic:  "test-topic.dlq",
	}
	err := validateConsumer(0, consumer)
	assert.NoError(t, err)
}

func TestValidateIndividualConsumers_MultipleConsumers(t *testing.T) {
	consumers := []ConsumerConfig{
		{Name: "consumer1", Topic: "topic1"},
		{Name: "consumer2", Topic: "topic2"},
		{Name: "consumer3", Topic: "topic3"},
	}

	err := validateIndividualConsumers(consumers)
	assert.NoError(t, err)
}

func TestValidateIndividualConsumers_OneInvalid(t *testing.T) {
	consumers := []ConsumerConfig{
		{Name: "consumer1", Topic: "topic1"},
		{Name: "", Topic: "topic2"}, // Invalid - empty name
		{Name: "consumer3", Topic: "topic3"},
	}

	err := validateIndividualConsumers(consumers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consumer[1]")
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestValidateProducerConfig_ReadinessTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		expectError bool
	}{
		{"zero (no timeout)", 0, false},
		{"valid timeout", 30, false},
		{"maximum timeout", maxReadinessTimeout, false},
		{"exceeds maximum", maxReadinessTimeout + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProducerConfig{
				ReadinessTimeoutSeconds: tt.timeout,
			}
			err := validateProducerConfig(cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "producer readiness timeout cannot exceed")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfig_CompleteInvalidConfig(t *testing.T) {
	cfg := &Config{
		Brokers: "",
		SchemaRegistry: SchemaRegistryConfig{
			URL:           "",
			CacheCapacity: 50,
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	// Should fail on first validation (brokers)
	assert.Contains(t, err.Error(), "brokers cannot be empty")
}

func TestValidateConfig_CompleteValidConfig(t *testing.T) {
	cfg := &Config{
		Brokers: "localhost:9092,localhost:9093",
		SchemaRegistry: SchemaRegistryConfig{
			URL:           "http://schema-registry:8081",
			CacheCapacity: 2000,
		},
		ConsumersConfig: ConsumersConfig{
			DefaultGroupID:           "test-group",
			DefaultAutoOffsetReset:   "latest",
			DefaultMaxRetries:        uintPtr(5),
			DefaultInitialBackoff:    2 * time.Second,
			DefaultMaxBackoff:        1 * time.Minute,
			DefaultChannelBufferSize: 200,
			ConsumerConfig: []ConsumerConfig{
				{
					Name:                    "consumer1",
					Topic:                   "topic1",
					GroupID:                 "group1",
					AutoOffsetReset:         "earliest",
					EnableDLQ:               true,
					DLQTopic:                "topic1.dlq",
					ReadinessTimeoutSeconds: 120,
					MaxRetries:              uintPtr(10),
					InitialBackoff:          500 * time.Millisecond,
					MaxBackoff:              2 * time.Minute,
					ChannelBufferSize:       500,
				},
			},
		},
		ProducerConfig: ProducerConfig{
			ReadinessTimeoutSeconds: 60,
			FailOnBrokerError:       true,
		},
	}

	err := validateConfig(cfg)
	assert.NoError(t, err)
}
