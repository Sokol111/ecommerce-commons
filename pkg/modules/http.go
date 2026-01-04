package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
)

// NewHTTPModule provides HTTP middleware functionality.
// Note: Server and health routes should be set up separately when using ogen.
func NewHTTPModule() fx.Option {
	return fx.Options(
		server.NewHTTPServerModule(),
		health.NewHealthRoutesModule(),
		middleware.NewOgenMiddlewareModule(),
	)
}
