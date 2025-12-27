package client

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DefaultTimeout is the default timeout for HTTP clients (suitable for internal service calls)
const DefaultTimeout = 10 * time.Second

// Config holds HTTP client configuration
type Config struct {
	// Timeout specifies a time limit for requests made by this Client
	Timeout time.Duration
	// Transport specifies the mechanism by which individual HTTP requests are made
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper
}

// NewHTTPClient creates a new HTTP client with OpenTelemetry instrumentation
func NewHTTPClient(opts ...Option) *http.Client {
	cfg := &Config{
		Timeout:   DefaultTimeout,
		Transport: http.DefaultTransport,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: otelhttp.NewTransport(cfg.Transport),
	}
}

// Option configures the HTTP client
type Option func(*Config)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithTransport sets the base transport (before otelhttp wrapping)
func WithTransport(transport http.RoundTripper) Option {
	return func(c *Config) {
		c.Transport = transport
	}
}
