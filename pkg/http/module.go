package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor"
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
)

// httpOptions holds internal configuration for the HTTP module.
type httpOptions struct {
	serverConfig *server.Config
}

// HTTPOption is a functional option for configuring the HTTP module.
type HTTPOption func(*httpOptions)

// WithServerConfig provides a static server Config (useful for tests).
// When set, the server configuration will not be loaded from koanf.
func WithServerConfig(cfg server.Config) HTTPOption {
	return func(opts *httpOptions) {
		opts.serverConfig = &cfg
	}
}

// WithH2C enables HTTP/2 without TLS (required for native gRPC support).
func WithH2C() HTTPOption {
	return func(opts *httpOptions) {
		if opts.serverConfig == nil {
			opts.serverConfig = &server.Config{}
		}
		opts.serverConfig.H2C = true
	}
}

// NewHTTPModule provides HTTP middleware functionality.
// It includes server, error handler, health routes, and middleware components.
//
// Options:
//   - WithServerConfig: provide static server Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from koanf
//	modules.NewHTTPModule()
//
//	// Testing - with static config
//	modules.NewHTTPModule(
//	    modules.WithServerConfig(server.Config{...}),
//	)
func NewHTTPModule(opts ...HTTPOption) fx.Option {
	cfg := &httpOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		serverModule(cfg),
		health.NewHealthRoutesModule(),
		interceptor.NewModule(),
	)
}

func serverModule(cfg *httpOptions) fx.Option {
	if cfg.serverConfig != nil {
		return server.NewHTTPServerModule(server.WithServerConfig(*cfg.serverConfig))
	}
	return server.NewHTTPServerModule()
}
