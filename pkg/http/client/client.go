package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/samber/lo"
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

func newHTTPClient(cfg Config) *http.Client {
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

	return &http.Client{
		Timeout:   timeout,
		Transport: &tenantTransport{base: retryTransport},
	}
}

// New creates an HTTP client from config, validating it and applying defaults.
func New(cfg Config) (*http.Client, error) {
	normalizedCfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	return newHTTPClient(normalizedCfg), nil
}

// LoadConfig loads an HTTP client config from the given koanf path.
func LoadConfig(k *koanf.Koanf, path string) (Config, error) {
	var clientCfg Config
	if err := k.Unmarshal(path, &clientCfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal client config %q: %w", path, err)
	}

	normalizedCfg, err := normalizeConfig(clientCfg)
	if err != nil {
		return Config{}, fmt.Errorf("invalid client config %q: %w", path, err)
	}

	return normalizedCfg, nil
}

// ProvideHTTPClient returns a provider function that creates an HTTP client from config
// Usage with fx:
//
//	fx.Provide(fx.Private, httpclient.ProvideHTTPClient("catalog-service"))
func ProvideHTTPClient(name string) func(*koanf.Koanf) (*http.Client, Config, error) {
	return func(k *koanf.Koanf) (*http.Client, Config, error) {
		clientCfg, err := LoadConfig(k, "clients."+name)
		if err != nil {
			return nil, Config{}, err
		}

		client, err := New(clientCfg)
		if err != nil {
			return nil, Config{}, err
		}

		return client, clientCfg, nil
	}
}

func normalizeConfig(cfg Config) (Config, error) {
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Timeout == nil {
		c.Timeout = lo.ToPtr(DefaultTimeout)
	}
	if c.MaxIdleConnsPerHost == nil {
		c.MaxIdleConnsPerHost = lo.ToPtr(DefaultMaxIdleConnsPerHost)
	}
	if c.IdleConnTimeout == nil {
		c.IdleConnTimeout = lo.ToPtr(DefaultIdleConnTimeout)
	}
	if c.MaxConnLifetime == nil {
		c.MaxConnLifetime = lo.ToPtr(DefaultMaxConnLifetime)
	}
}

func (c Config) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base-url is required")
	}
	return nil
}
