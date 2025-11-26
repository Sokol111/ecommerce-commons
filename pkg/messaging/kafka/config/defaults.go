package config

// applyDefaults applies default values to the configuration
func applyDefaults(cfg *Config) {
	// Apply defaults for schema registry
	if cfg.SchemaRegistry.CacheCapacity == 0 {
		cfg.SchemaRegistry.CacheCapacity = defaultSchemaRegistryCacheCapacity
	}

	// Apply defaults for global consumer config
	if cfg.ConsumersConfig.DefaultMaxRetryAttempts == 0 {
		cfg.ConsumersConfig.DefaultMaxRetryAttempts = defaultMaxRetryAttempts
	}
	if cfg.ConsumersConfig.DefaultInitialBackoff == 0 {
		cfg.ConsumersConfig.DefaultInitialBackoff = defaultInitialBackoff
	}
	if cfg.ConsumersConfig.DefaultMaxBackoff == 0 {
		cfg.ConsumersConfig.DefaultMaxBackoff = defaultMaxBackoff
	}
	if cfg.ConsumersConfig.DefaultChannelBufferSize == 0 {
		cfg.ConsumersConfig.DefaultChannelBufferSize = defaultChannelBufferSize
	}

	// Apply defaults from global consumer config to individual consumers
	for i := range cfg.ConsumersConfig.ConsumerConfig {
		applyConsumerDefaults(&cfg.ConsumersConfig.ConsumerConfig[i], &cfg.ConsumersConfig)
	}

	// Apply default producer config settings
	if cfg.ProducerConfig.ReadinessTimeoutSeconds == 0 {
		cfg.ProducerConfig.ReadinessTimeoutSeconds = defaultProducerReadinessTimeout
	}
}

// applyConsumerDefaults applies defaults to an individual consumer configuration
func applyConsumerDefaults(consumer *ConsumerConfig, globalConfig *ConsumersConfig) {
	if consumer.GroupID == "" {
		consumer.GroupID = globalConfig.DefaultGroupID
	}
	if consumer.AutoOffsetReset == "" {
		consumer.AutoOffsetReset = globalConfig.DefaultAutoOffsetReset
	}
	// Apply default DLQ topic naming convention: {topic}.dlq
	if consumer.EnableDLQ && consumer.DLQTopic == "" {
		consumer.DLQTopic = consumer.Topic + ".dlq"
	}
	// Apply default readiness timeout
	if consumer.ReadinessTimeoutSeconds == 0 {
		consumer.ReadinessTimeoutSeconds = defaultConsumerReadinessTimeout
	}
	// Apply default max retry attempts from global config
	if consumer.MaxRetryAttempts == 0 {
		consumer.MaxRetryAttempts = globalConfig.DefaultMaxRetryAttempts
	}
	// Apply default initial backoff from global config
	if consumer.InitialBackoff == 0 {
		consumer.InitialBackoff = globalConfig.DefaultInitialBackoff
	}
	// Apply default max backoff from global config
	if consumer.MaxBackoff == 0 {
		consumer.MaxBackoff = globalConfig.DefaultMaxBackoff
	}
	// Apply default channel buffer size from global config
	if consumer.ChannelBufferSize == 0 {
		consumer.ChannelBufferSize = globalConfig.DefaultChannelBufferSize
	}
}
