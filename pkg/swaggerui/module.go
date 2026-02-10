package swaggerui

import (
	"net/http"

	"go.uber.org/fx"
)

// NewSwaggerModule provides Swagger UI endpoints for API documentation.
func NewSwaggerModule(cfg SwaggerConfig) fx.Option {
	return fx.Invoke(func(mux *http.ServeMux) {
		registerSwaggerUI(mux, cfg)
	})
}
