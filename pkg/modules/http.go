package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
)

// NewHTTPModule provides HTTP functionality: gin, server, health, timeout, rate limiting, bulkhead.
func NewHTTPModule() fx.Option {
	return fx.Options(
		middleware.NewGinModule(),
		server.NewHTTPServerModule(),
		health.NewHealthRoutesModule(),
	)
}
