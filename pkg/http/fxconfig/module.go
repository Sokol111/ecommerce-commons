package fxconfig

import (
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	fx_health "github.com/Sokol111/ecommerce-commons/pkg/http/health/fxconfig"
	fx_server "github.com/Sokol111/ecommerce-commons/pkg/http/server/fxconfig"
	"go.uber.org/fx"
)

// httpOptions holds internal configuration for the HTTP module.
type httpOptions struct {
	serverOpts []fx_server.Option
}

// Option is a functional option for configuring the HTTP module.
type Option func(*httpOptions)

// WithServerOptions passes options to the underlying server module.
func WithServerOptions(opts ...fx_server.Option) Option {
	return func(o *httpOptions) {
		o.serverOpts = append(o.serverOpts, opts...)
	}
}

// NewHTTPModule provides HTTP middleware functionality.
// It includes server, error handler, health routes, and middleware components.
func NewHTTPModule(opts ...Option) fx.Option {
	cfg := &httpOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx_server.NewHTTPServerModule(cfg.serverOpts...),
		fx_health.NewHealthRoutesModule(),
		fx_interceptor.NewInterceptorsModule(),
	)
}
