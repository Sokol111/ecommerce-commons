package health

import (
	"net/http"

	"go.uber.org/fx"
)

// NewHealthRoutesModule registers health endpoints on the ServeMux.
func NewHealthRoutesModule() fx.Option {
	return fx.Options(
		fx.Provide(newHealthHandler),
		fx.Invoke(registerHealthRoutes),
	)
}

func registerHealthRoutes(mux *http.ServeMux, h *healthHandler) {
	mux.HandleFunc("/health/ready", h.IsReady)
	mux.HandleFunc("/health/live", h.IsLive)
}
