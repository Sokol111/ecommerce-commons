package config

import (
	"fmt"
	"strings"
)

// validateConfig validates the entire Kafka configuration
func validateConfig(cfg *Config) error {
	if err := validateBrokers(cfg); err != nil {
		return err
	}
	if err := validateSchemaRegistry(&cfg.SchemaRegistry); err != nil {
		return err
	}
	if err := validateGlobalConsumerConfig(&cfg.ConsumersConfig); err != nil {
		return err
	}
	if err := validateIndividualConsumers(cfg.ConsumersConfig.ConsumerConfig); err != nil {
		return err
	}
	if err := validateProducerConfig(&cfg.ProducerConfig); err != nil {
		return err
	}
	return nil
}

// validateBrokers validates Kafka brokers configuration
func validateBrokers(cfg *Config) error {
	if strings.TrimSpace(cfg.Brokers) == "" {
		return fmt.Errorf("kafka brokers cannot be empty")
	}
	return nil
}

// validateSchemaRegistry validates Schema Registry configuration
func validateSchemaRegistry(cfg *SchemaRegistryConfig) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("schema registry URL cannot be empty")
	}
	if cfg.CacheCapacity > 0 && (cfg.CacheCapacity < minSchemaCacheCapacity || cfg.CacheCapacity > maxSchemaCacheCapacity) {
		return fmt.Errorf("schema registry cache capacity must be between %d and %d, got: %d",
			minSchemaCacheCapacity, maxSchemaCacheCapacity, cfg.CacheCapacity)
	}
	return nil
}

// validateGlobalConsumerConfig validates global consumer configuration
func validateGlobalConsumerConfig(cfg *ConsumersConfig) error {
	if cfg.DefaultMaxRetryAttempts > 0 &&
		(cfg.DefaultMaxRetryAttempts < minMaxRetryAttempts || cfg.DefaultMaxRetryAttempts > maxMaxRetryAttempts) {
		return fmt.Errorf("default max retry attempts must be between %d and %d, got: %d",
			minMaxRetryAttempts, maxMaxRetryAttempts, cfg.DefaultMaxRetryAttempts)
	}
	if cfg.DefaultInitialBackoff > 0 &&
		(cfg.DefaultInitialBackoff < minInitialBackoff || cfg.DefaultInitialBackoff > maxInitialBackoff) {
		return fmt.Errorf("default initial backoff must be between %v and %v, got: %v",
			minInitialBackoff, maxInitialBackoff, cfg.DefaultInitialBackoff)
	}
	if cfg.DefaultMaxBackoff > 0 &&
		(cfg.DefaultMaxBackoff < minMaxBackoff || cfg.DefaultMaxBackoff > maxMaxBackoffDuration) {
		return fmt.Errorf("default max backoff must be between %v and %v, got: %v",
			minMaxBackoff, maxMaxBackoffDuration, cfg.DefaultMaxBackoff)
	}
	if cfg.DefaultMaxBackoff > 0 && cfg.DefaultInitialBackoff > cfg.DefaultMaxBackoff {
		return fmt.Errorf("default initial backoff (%v) cannot be greater than max backoff (%v)",
			cfg.DefaultInitialBackoff, cfg.DefaultMaxBackoff)
	}
	if cfg.DefaultProcessingTimeout > 0 &&
		(cfg.DefaultProcessingTimeout < minProcessingTimeout || cfg.DefaultProcessingTimeout > maxProcessingTimeout) {
		return fmt.Errorf("default processing timeout must be between %v and %v, got: %v",
			minProcessingTimeout, maxProcessingTimeout, cfg.DefaultProcessingTimeout)
	}
	if cfg.DefaultChannelBufferSize > 0 &&
		(cfg.DefaultChannelBufferSize < minChannelBufferSize || cfg.DefaultChannelBufferSize > maxChannelBufferSize) {
		return fmt.Errorf("default channel buffer size must be between %d and %d, got: %d",
			minChannelBufferSize, maxChannelBufferSize, cfg.DefaultChannelBufferSize)
	}
	return nil
}

// validateIndividualConsumers validates all individual consumer configurations
func validateIndividualConsumers(consumers []ConsumerConfig) error {
	for i, consumer := range consumers {
		if err := validateConsumer(i, &consumer); err != nil {
			return err
		}
	}
	return nil
}

// validateConsumer validates a single consumer configuration
func validateConsumer(index int, consumer *ConsumerConfig) error {
	if strings.TrimSpace(consumer.Name) == "" {
		return fmt.Errorf("consumer[%d]: name cannot be empty", index)
	}
	if strings.TrimSpace(consumer.Topic) == "" {
		return fmt.Errorf("consumer[%d] (%s): topic cannot be empty", index, consumer.Name)
	}
	if consumer.AutoOffsetReset != "" && consumer.AutoOffsetReset != "earliest" && consumer.AutoOffsetReset != "latest" {
		return fmt.Errorf("consumer[%d] (%s): auto offset reset must be 'earliest' or 'latest', got: %s",
			index, consumer.Name, consumer.AutoOffsetReset)
	}
	if consumer.ReadinessTimeoutSeconds > maxReadinessTimeout {
		return fmt.Errorf("consumer[%d] (%s): readiness timeout cannot exceed %d seconds, got: %d",
			index, consumer.Name, maxReadinessTimeout, consumer.ReadinessTimeoutSeconds)
	}
	if consumer.MaxRetryAttempts > 0 &&
		(consumer.MaxRetryAttempts < minMaxRetryAttempts || consumer.MaxRetryAttempts > maxMaxRetryAttempts) {
		return fmt.Errorf("consumer[%d] (%s): max retry attempts must be between %d and %d, got: %d",
			index, consumer.Name, minMaxRetryAttempts, maxMaxRetryAttempts, consumer.MaxRetryAttempts)
	}
	if consumer.InitialBackoff > 0 &&
		(consumer.InitialBackoff < minInitialBackoff || consumer.InitialBackoff > maxInitialBackoff) {
		return fmt.Errorf("consumer[%d] (%s): initial backoff must be between %v and %v, got: %v",
			index, consumer.Name, minInitialBackoff, maxInitialBackoff, consumer.InitialBackoff)
	}
	if consumer.MaxBackoff > 0 &&
		(consumer.MaxBackoff < minMaxBackoff || consumer.MaxBackoff > maxMaxBackoffDuration) {
		return fmt.Errorf("consumer[%d] (%s): max backoff must be between %v and %v, got: %v",
			index, consumer.Name, minMaxBackoff, maxMaxBackoffDuration, consumer.MaxBackoff)
	}
	if consumer.MaxBackoff > 0 && consumer.InitialBackoff > consumer.MaxBackoff {
		return fmt.Errorf("consumer[%d] (%s): initial backoff (%v) cannot be greater than max backoff (%v)",
			index, consumer.Name, consumer.InitialBackoff, consumer.MaxBackoff)
	}
	if consumer.ProcessingTimeout > 0 &&
		(consumer.ProcessingTimeout < minProcessingTimeout || consumer.ProcessingTimeout > maxProcessingTimeout) {
		return fmt.Errorf("consumer[%d] (%s): processing timeout must be between %v and %v, got: %v",
			index, consumer.Name, minProcessingTimeout, maxProcessingTimeout, consumer.ProcessingTimeout)
	}
	if consumer.ChannelBufferSize > 0 &&
		(consumer.ChannelBufferSize < minChannelBufferSize || consumer.ChannelBufferSize > maxChannelBufferSize) {
		return fmt.Errorf("consumer[%d] (%s): channel buffer size must be between %d and %d, got: %d",
			index, consumer.Name, minChannelBufferSize, maxChannelBufferSize, consumer.ChannelBufferSize)
	}
	if consumer.EnableDLQ && strings.TrimSpace(consumer.DLQTopic) != "" && consumer.DLQTopic == consumer.Topic {
		return fmt.Errorf("consumer[%d] (%s): DLQ topic cannot be the same as main topic",
			index, consumer.Name)
	}
	return nil
}

// validateProducerConfig validates producer configuration
func validateProducerConfig(cfg *ProducerConfig) error {
	if cfg.ReadinessTimeoutSeconds > maxReadinessTimeout {
		return fmt.Errorf("producer readiness timeout cannot exceed %d seconds, got: %d",
			maxReadinessTimeout, cfg.ReadinessTimeoutSeconds)
	}
	return nil
}
