package config

import "time"

const (
	// Default values.
	defaultSchemaRegistryCacheCapacity = 1000
	defaultMaxRetryAttempts            = 3
	defaultInitialBackoff              = 1 * time.Second
	defaultMaxBackoff                  = 30 * time.Second
	defaultProcessingTimeout           = 30 * time.Second
	defaultChannelBufferSize           = 100
	defaultConsumerReadinessTimeout    = 60
	defaultProducerReadinessTimeout    = 30

	// Validation bounds.
	minMaxRetryAttempts    = 1
	maxMaxRetryAttempts    = 100
	minInitialBackoff      = 100 * time.Millisecond
	maxInitialBackoff      = 30 * time.Second
	minMaxBackoff          = 1 * time.Second
	maxMaxBackoffDuration  = 5 * time.Minute
	minProcessingTimeout   = 1 * time.Second
	maxProcessingTimeout   = 10 * time.Minute
	minChannelBufferSize   = 10
	maxChannelBufferSize   = 10000
	minSchemaCacheCapacity = 100
	maxSchemaCacheCapacity = 100000
	maxReadinessTimeout    = 600 // 10 minutes in seconds
)
