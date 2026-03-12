package config

import "time"

const (
	// Default values.
	defaultSchemaRegistryCacheCapacity = 1000
	defaultMaxRetries                  = uint(2)
	defaultInitialBackoff              = 1 * time.Second
	defaultMaxBackoff                  = 30 * time.Second
	defaultProcessingTimeout           = 30 * time.Second
	defaultChannelBufferSize           = 100
	defaultMaxPollRecords              = 500
	defaultConsumerReadinessTimeout    = 60
	defaultProducerReadinessTimeout    = 30
	defaultProducerLinger              = 5 * time.Millisecond
	defaultProducerCompression         = "none"
	defaultProducerDeliveryTimeout     = 30 * time.Second
	defaultProducerMaxBufferedRecords  = 10000

	// Validation bounds.
	minMaxRetries          = uint(0)
	maxMaxRetries          = uint(99)
	minInitialBackoff      = 100 * time.Millisecond
	maxInitialBackoff      = 30 * time.Second
	minMaxBackoff          = 1 * time.Second
	maxMaxBackoffDuration  = 5 * time.Minute
	minProcessingTimeout   = 1 * time.Second
	maxProcessingTimeout   = 10 * time.Minute
	minChannelBufferSize   = 10
	maxChannelBufferSize   = 10000
	minMaxPollRecords      = 1
	maxMaxPollRecords      = 10000
	minSchemaCacheCapacity = 100
	maxSchemaCacheCapacity = 100000
	maxReadinessTimeout    = 600 // 10 minutes in seconds

	// Producer bounds.
	maxProducerLinger             = 1 * time.Second
	minProducerDeliveryTimeout    = 1 * time.Second
	maxProducerDeliveryTimeout    = 5 * time.Minute
	minProducerMaxBufferedRecords = 100
	maxProducerMaxBufferedRecords = 1000000
)
