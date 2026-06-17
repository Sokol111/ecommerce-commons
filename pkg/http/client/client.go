package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// Default values for HTTP client configuration.
// Optimized for K8s: MaxConnLifetime ensures rebalancing, so pool can be larger for better performance.
const (
	DefaultTimeout             = 10 * time.Second
	DefaultMaxIdleConnsPerHost = 100              // Larger pool for connection reuse; rebalancing handled by MaxConnLifetime
	DefaultIdleConnTimeout     = 90 * time.Second // Keep idle connections longer; lifetime-based rotation handles freshness
	DefaultMaxConnLifetime     = 60 * time.Second // Force connection refresh for load balancing to new pods
	MaxRetriesCap              = 5                // Retries before pool reset; 97% success rate with 50% bad connections
)

// Config holds configuration for an HTTP client loaded from config file.
// yaml example:
//
//	clients:
//	  catalog-service:
//	    base-url: http://catalog-service:8080
//	    timeout: 10s
//	    max-idle-conns-per-host: 10
//	    idle-conn-timeout: 10s
//	    max-conn-lifetime: 60s
//
// Omit timeout fields to use defaults. Set to 0 to disable.
type Config struct {
	BaseURL             string         `koanf:"base-url"`
	Timeout             *time.Duration `koanf:"timeout"`
	MaxIdleConnsPerHost *int           `koanf:"max-idle-conns-per-host"`
	IdleConnTimeout     *time.Duration `koanf:"idle-conn-timeout"`
	MaxConnLifetime     *time.Duration `koanf:"max-conn-lifetime"`
}

func newHTTPClient(cfg Config, tokenSource oauth2.TokenSource) *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	maxConnLifetime := *cfg.MaxConnLifetime
	idleConnTimeout := *cfg.IdleConnTimeout
	maxIdleConnsPerHost := *cfg.MaxIdleConnsPerHost
	timeout := *cfg.Timeout

	// Custom DialContext only needed if MaxConnLifetime is enabled
	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	if maxConnLifetime > 0 {
		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			// Wrap connection with max lifetime tracking
			return &timedConn{
				Conn:        conn,
				createdAt:   time.Now(),
				maxLifetime: maxConnLifetime,
			}, nil
		}
	}

	transport := &http.Transport{
		DialContext:         dialContext,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
	}

	// Wrap transport with retry logic for transient errors
	// Retries = min(pool size, cap) to exhaust dead connections without excessive attempts
	retryTransport := &retryTransport{
		base:       transport,
		transport:  transport,
		maxRetries: min(maxIdleConnsPerHost, MaxRetriesCap),
	}

	// Build transport chain: tenant (propagation) < auth (injection, optional) < retry < base
	chain := http.RoundTripper(retryTransport)
	chain = &tenantTransport{base: chain}

	if tokenSource != nil {
		chain = &authTransport{base: chain, token: tokenSource}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: chain,
	}
}

// ApplyDefaults fills in zero-value fields with the package-level defaults.
func (c *Config) ApplyDefaults() {
	if c.Timeout == nil {
		c.Timeout = new(DefaultTimeout)
	}
	if c.MaxIdleConnsPerHost == nil {
		c.MaxIdleConnsPerHost = new(DefaultMaxIdleConnsPerHost)
	}
	if c.IdleConnTimeout == nil {
		c.IdleConnTimeout = new(DefaultIdleConnTimeout)
	}
	if c.MaxConnLifetime == nil {
		c.MaxConnLifetime = new(DefaultMaxConnLifetime)
	}
}

// Validate returns an error if the Config is missing required fields.
func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base-url is required")
	}
	return nil
}
