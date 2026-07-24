package fxconfig

import (
	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"go.uber.org/fx"
)

// NewHealthRoutesModule registers health endpoints on the ServeMux.
func NewHealthRoutesModule() fx.Option {
	return fx.Options(
		fx.Provide(health.NewHealthHandler),
		fx.Invoke(health.RegisterHealthRoutes),
	)
}
