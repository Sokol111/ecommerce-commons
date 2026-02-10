package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
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
// When set, the server configuration will not be loaded from viper.
func WithServerConfig(cfg server.Config) HTTPOption {
	return func(opts *httpOptions) {
		opts.serverConfig = &cfg
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
//	// Production - loads config from viper
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
		problems.NewErrorHandlerModule(),
		health.NewHealthRoutesModule(),
		middleware.NewMiddlewareModule(),
	)
}

func serverModule(cfg *httpOptions) fx.Option {
	if cfg.serverConfig != nil {
		return server.NewHTTPServerModule(server.WithServerConfig(*cfg.serverConfig))
	}
	return server.NewHTTPServerModule()
}
