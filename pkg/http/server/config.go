package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	// writeTimeoutBuffer is added to RequestTimeout to calculate WriteTimeout.
	// This ensures the timeout middleware has time to send a proper HTTP response
	// before the connection is forcibly closed.
	writeTimeoutBuffer = 5 * time.Second

	// Maximum configuration values to prevent misconfiguration.
	maxReadHeaderTimeout = 1 * time.Minute  // 1 min - enough for slow clients
	maxReadTimeout       = 5 * time.Minute  // 5 min - for large uploads
	maxIdleTimeout       = 10 * time.Minute // 10 min - keep-alive shouldn't be too long
	maxRequestTimeout    = 5 * time.Minute  // 5 min - longer operations should be async
	maxHeaderBytes       = 10 << 20         // 10 MB - headers shouldn't be this large
	maxRequestsPerSecond = 1_000_000        // 1M req/s - sanity check
	maxBurst             = 100_000          // 100K burst - sanity check
	maxConcurrent        = 100_000          // 100K concurrent - sanity check
	maxBulkheadTimeout   = 1 * time.Minute  // 1 min - waiting longer is pointless
)

// Config holds the HTTP server configuration.
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

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests-per-second"`
	Burst             int  `mapstructure:"burst"`
}

// BulkheadConfig holds HTTP bulkhead (concurrency limiting) configuration.
type BulkheadConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	MaxConcurrent int           `mapstructure:"max-concurrent"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

// Validate validates the configuration values.
// Zero values are acceptable (will use defaults), but negative values are errors.
func (c *Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535, got %d", c.Port)
	}

	if err := c.Connection.validate(); err != nil {
		return fmt.Errorf("connection: %w", err)
	}
	if err := c.Timeout.validate(); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	if err := c.RateLimit.validate(); err != nil {
		return fmt.Errorf("rate-limit: %w", err)
	}
	if err := c.Bulkhead.validate(); err != nil {
		return fmt.Errorf("bulkhead: %w", err)
	}

	return nil
}

func (c *ConnectionConfig) validate() error {
	if c.ReadHeaderTimeout < 0 {
		return errors.New("read-header-timeout cannot be negative")
	}
	if c.ReadHeaderTimeout > maxReadHeaderTimeout {
		return fmt.Errorf("read-header-timeout cannot exceed %v", maxReadHeaderTimeout)
	}
	if c.ReadTimeout < 0 {
		return errors.New("read-timeout cannot be negative")
	}
	if c.ReadTimeout > maxReadTimeout {
		return fmt.Errorf("read-timeout cannot exceed %v", maxReadTimeout)
	}
	if c.IdleTimeout < 0 {
		return errors.New("idle-timeout cannot be negative")
	}
	if c.IdleTimeout > maxIdleTimeout {
		return fmt.Errorf("idle-timeout cannot exceed %v", maxIdleTimeout)
	}
	if c.MaxHeaderBytes < 0 {
		return errors.New("max-header-bytes cannot be negative")
	}
	if c.MaxHeaderBytes > maxHeaderBytes {
		return fmt.Errorf("max-header-bytes cannot exceed %d", maxHeaderBytes)
	}
	return nil
}

func (c *TimeoutConfig) validate() error {
	if c.RequestTimeout < 0 {
		return errors.New("request-timeout cannot be negative")
	}
	if c.RequestTimeout > maxRequestTimeout {
		return fmt.Errorf("request-timeout cannot exceed %v", maxRequestTimeout)
	}
	return nil
}

func (c *RateLimitConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if c.RequestsPerSecond < 0 {
		return errors.New("requests-per-second cannot be negative")
	}
	if c.RequestsPerSecond > maxRequestsPerSecond {
		return fmt.Errorf("requests-per-second cannot exceed %d", maxRequestsPerSecond)
	}
	if c.Burst < 0 {
		return errors.New("burst cannot be negative")
	}
	if c.Burst > maxBurst {
		return fmt.Errorf("burst cannot exceed %d", maxBurst)
	}
	return nil
}

func (c *BulkheadConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if c.MaxConcurrent < 0 {
		return errors.New("max-concurrent cannot be negative")
	}
	if c.MaxConcurrent > maxConcurrent {
		return fmt.Errorf("max-concurrent cannot exceed %d", maxConcurrent)
	}
	if c.Timeout < 0 {
		return errors.New("timeout cannot be negative")
	}
	if c.Timeout > maxBulkheadTimeout {
		return fmt.Errorf("timeout cannot exceed %v", maxBulkheadTimeout)
	}
	return nil
}

func loadConfigFromViper(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("server").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}
	return cfg, nil
}

// setDefaults sets default values for all configuration sections.
func (c *Config) setDefaults() {
	if c.Port == 0 {
		c.Port = 8080 // Default: 8080
	}
	c.Connection.setDefaults(c.Timeout)
	c.RateLimit.setDefaults()
	c.Bulkhead.setDefaults()
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
