package swaggerui

import (
	"net/http"

	"go.uber.org/fx"
)

// NewSwaggerModule provides Swagger UI endpoints for API documentation.
// SwaggerConfig is resolved from the fx dependency injection container.
func NewSwaggerModule() fx.Option {
	return fx.Invoke(func(mux *http.ServeMux, cfg SwaggerConfig) {
		registerSwaggerUI(mux, cfg)
	})
}
