package server

import (
	"context"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewHTTPServerModule provides HTTP server components for dependency injection.
func NewHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(newConfig),
		fx.Provide(newServeMux),
		fx.Invoke(startHTTPServer),
	)
}

func newServeMux() (*http.ServeMux, http.Handler) {
	mux := http.NewServeMux()
	return mux, mux
}

func startHTTPServer(lc fx.Lifecycle, log *zap.Logger, conf Config, handler http.Handler, readiness health.ComponentManager, shutdowner fx.Shutdowner) {
	var srv Server
	markReady := readiness.AddComponent("http-server")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create server in OnStart - all routes are registered by now
			srv = newServer(log, conf, handler)

			go func() {
				if err := srv.ServeWithReadyCallback(func() {
					markReady()
				}); err != nil {
					log.Error("HTTP server failed, shutting down application", zap.Error(err))
					_ = shutdowner.Shutdown() //nolint:errcheck // shutdown is best-effort
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if srv != nil {
				return srv.Shutdown(ctx)
			}
			return nil
		},
	})
}
