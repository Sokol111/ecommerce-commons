package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
)

// Default values for HTTP client configuration
// Optimized for K8s: MaxConnLifetime ensures rebalancing, so pool can be larger for better performance
const (
	DefaultTimeout             = 10 * time.Second
	DefaultMaxIdleConnsPerHost = 100              // Larger pool for connection reuse; rebalancing handled by MaxConnLifetime
	DefaultIdleConnTimeout     = 90 * time.Second // Keep idle connections longer; lifetime-based rotation handles freshness
	DefaultMaxConnLifetime     = 60 * time.Second // Force connection refresh for load balancing to new pods
	MaxRetriesCap              = 5                // Retries before pool reset; 97% success rate with 50% bad connections
)

// ClientConfig holds configuration for an HTTP client loaded from config file
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
type ClientConfig struct {
	BaseURL             string         `mapstructure:"base-url"`
	Timeout             *time.Duration `mapstructure:"timeout"`
	MaxIdleConnsPerHost *int           `mapstructure:"max-idle-conns-per-host"`
	IdleConnTimeout     *time.Duration `mapstructure:"idle-conn-timeout"`
	MaxConnLifetime     *time.Duration `mapstructure:"max-conn-lifetime"`
}

func newHTTPClient(cfg ClientConfig) *http.Client {
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
		Transport: retryTransport,
	}
}

// ProvideHTTPClient returns a provider function that creates an HTTP client from config
// Usage with fx:
//
//	fx.Provide(fx.Private, httpclient.ProvideHTTPClient("catalog-service"))
func ProvideHTTPClient(name string) func(*viper.Viper) (*http.Client, ClientConfig, error) {
	return func(cfg *viper.Viper) (*http.Client, ClientConfig, error) {
		var clientCfg ClientConfig
		if err := cfg.UnmarshalKey("clients."+name, &clientCfg); err != nil {
			return nil, ClientConfig{}, fmt.Errorf("failed to unmarshal client config %q: %w", name, err)
		}
		if err := clientCfg.validate(); err != nil {
			return nil, ClientConfig{}, fmt.Errorf("invalid client config %q: %w", name, err)
		}
		clientCfg.applyDefaults()
		return newHTTPClient(clientCfg), clientCfg, nil
	}
}

func (c *ClientConfig) applyDefaults() {
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

func (c ClientConfig) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base-url is required")
	}
	return nil
}
