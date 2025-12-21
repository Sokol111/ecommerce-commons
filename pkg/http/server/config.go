package server

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	// writeTimeoutBuffer is added to RequestTimeout to calculate WriteTimeout.
	// This ensures the timeout middleware has time to send a proper HTTP response
	// before the connection is forcibly closed.
	writeTimeoutBuffer = 5 * time.Second
)

type Config struct {
	Port int `mapstructure:"port"`

	// Server connection settings
	Connection ConnectionConfig `mapstructure:"connection"`

	// Request Timeout (middleware-based, returns proper HTTP response)
	Timeout TimeoutConfig `mapstructure:"timeout"`

	// Rate Limiting
	RateLimit RateLimitConfig `mapstructure:"rate-limit"`

	// HTTP Bulkhead
	Bulkhead BulkheadConfig `mapstructure:"bulkhead"`

	// Circuit Breaker
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit-breaker"`
}

// ConnectionConfig contains low-level HTTP server connection settings.
// These are "hard" timeouts that close the connection without HTTP response.
//
// Note: WriteTimeout is intentionally not configurable. It is automatically
// calculated as RequestTimeout + buffer to ensure the timeout middleware
// can send a proper HTTP response before the connection is closed.
type ConnectionConfig struct {
	ReadHeaderTimeout time.Duration `mapstructure:"read-header-timeout"` // Time to read request headers (Slowloris protection)
	ReadTimeout       time.Duration `mapstructure:"read-timeout"`        // Time to read entire request (headers + body)
	WriteTimeout      time.Duration `mapstructure:"-"`                   // Auto-calculated, not configurable
	IdleTimeout       time.Duration `mapstructure:"idle-timeout"`        // Keep-alive timeout between requests
	MaxHeaderBytes    int           `mapstructure:"max-header-bytes"`    // Max size of request headers
}

// TimeoutConfig controls the middleware-based request timeout.
// Unlike connection timeouts, this returns a proper HTTP 503 response.
// Timeout is enabled when RequestTimeout > 0.
type TimeoutConfig struct {
	RequestTimeout time.Duration `mapstructure:"request-timeout"` // Max time to handle a request (0 = disabled)
}

type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests-per-second"`
	Burst             int  `mapstructure:"burst"`
}

type BulkheadConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	MaxConcurrent int           `mapstructure:"max-concurrent"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	FailureThreshold uint32        `mapstructure:"failure-threshold"`
	Timeout          time.Duration `mapstructure:"timeout"`
	Interval         time.Duration `mapstructure:"interval"`
	MaxRequests      uint32        `mapstructure:"max-requests"`
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("server").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}

	cfg.Connection.setDefaults(cfg.Timeout)
	cfg.RateLimit.setDefaults()
	cfg.Bulkhead.setDefaults()
	cfg.CircuitBreaker.setDefaults()

	logger.Info("loaded server config", zap.Any("config", cfg))
	return cfg, nil
}

// setDefaults sets default values for server connection settings (optimized for API services).
func (c *ConnectionConfig) setDefaults(timeout TimeoutConfig) {
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = 10 * time.Second // Default: 10s - protection against Slowloris
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 30 * time.Second // Default: 30s - enough for typical API requests
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 120 * time.Second // Default: 120s - keep-alive timeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = 1 << 20 // Default: 1 MB
	}

	// WriteTimeout is derived from RequestTimeout when timeout middleware is enabled.
	// It must be larger than RequestTimeout so the middleware can send an HTTP response.
	// When timeout middleware is disabled, WriteTimeout is also disabled (0).
	if timeout.RequestTimeout > 0 {
		c.WriteTimeout = timeout.RequestTimeout + writeTimeoutBuffer
	}
}

// setDefaults sets default values for rate limiting configuration.
func (c *RateLimitConfig) setDefaults() {
	if !c.Enabled {
		return
	}
	if c.RequestsPerSecond <= 0 {
		c.RequestsPerSecond = 1000 // Default: 1000 req/s
	}
	if c.Burst <= 0 {
		c.Burst = 100 // Default: burst of 100
	}
}

// setDefaults sets default values for HTTP bulkhead configuration.
func (c *BulkheadConfig) setDefaults() {
	if !c.Enabled {
		return
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 500 // Default: 500 concurrent requests
	}
	if c.Timeout <= 0 {
		c.Timeout = 100 * time.Millisecond // Default: 100ms timeout
	}
}

// setDefaults sets default values for circuit breaker configuration.
func (c *CircuitBreakerConfig) setDefaults() {
	if !c.Enabled {
		return
	}
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 5 // Default: 5 consecutive failures
	}
	if c.Timeout <= 0 {
		c.Timeout = 60 * time.Second // Default: 60 seconds
	}
	if c.Interval <= 0 {
		c.Interval = 60 * time.Second // Default: 60 seconds (reset interval)
	}
	if c.MaxRequests <= 0 {
		c.MaxRequests = 1 // Default: 1 request in half-open state
	}
}
