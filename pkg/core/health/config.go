package health

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	RunningInKubernetes bool `mapstructure:"running-in-kubernetes"`
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("readiness").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}

	// Set default values for timeout
	if cfg.Timeout.Enabled && cfg.Timeout.RequestTimeout == 0 {
		cfg.Timeout.RequestTimeout = 30 * time.Second // Default: 30 seconds
	}

	// Set default values for rate limiting
	if cfg.RateLimit.Enabled && cfg.RateLimit.RequestsPerSecond == 0 {
		cfg.RateLimit.RequestsPerSecond = 1000 // Default: 1000 req/s
	}
	if cfg.RateLimit.Enabled && cfg.RateLimit.Burst == 0 {
		cfg.RateLimit.Burst = 100 // Default: burst of 100
	}

	// Set default values for HTTP bulkhead
	if cfg.Bulkhead.Enabled && cfg.Bulkhead.MaxConcurrent == 0 {
		cfg.Bulkhead.MaxConcurrent = 500 // Default: 500 concurrent requests
	}
	if cfg.Bulkhead.Enabled && cfg.Bulkhead.Timeout == 0 {
		cfg.Bulkhead.Timeout = 100 * time.Millisecond // Default: 100ms timeout
	}

	// Set default values for circuit breaker
	if cfg.CircuitBreaker.Enabled && cfg.CircuitBreaker.FailureThreshold == 0 {
		cfg.CircuitBreaker.FailureThreshold = 5 // Default: 5 consecutive failures
	}
	if cfg.CircuitBreaker.Enabled && cfg.CircuitBreaker.Timeout == 0 {
		cfg.CircuitBreaker.Timeout = 60 * time.Second // Default: 60 seconds
	}
	if cfg.CircuitBreaker.Enabled && cfg.CircuitBreaker.Interval == 0 {
		cfg.CircuitBreaker.Interval = 60 * time.Second // Default: 60 seconds (reset interval)
	}
	if cfg.CircuitBreaker.Enabled && cfg.CircuitBreaker.MaxRequests == 0 {
		cfg.CircuitBreaker.MaxRequests = 1 // Default: 1 request in half-open state
	}

	logger.Info("loaded server config", zap.Any("config", cfg))
	return cfg, nil
}
