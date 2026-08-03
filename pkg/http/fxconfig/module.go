package fxconfig

import (
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	fx_health "github.com/Sokol111/ecommerce-commons/pkg/http/health/fxconfig"
	fx_server "github.com/Sokol111/ecommerce-commons/pkg/http/server/fxconfig"
	"go.uber.org/fx"
)

// NewHTTPModule provides HTTP middleware functionality.
// It includes server, error handler, health routes, and middleware components.
func NewHTTPModule() fx.Option {
	return fx.Options(
		fx_server.NewHTTPServerModule(),
		fx_health.NewHealthRoutesModule(),
		fx_interceptor.NewInterceptorsModule(),
	)
}
