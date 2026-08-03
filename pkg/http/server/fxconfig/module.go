package fxconfig

import (
	"context"
	"net"
	"net/http"
	"strconv"

	coreconf "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewHTTPServerModule provides HTTP server components for dependency injection.
// By default, configuration is loaded from koanf.
func NewHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
		fx.Provide(func() (*http.ServeMux, http.Handler) {
			mux := http.NewServeMux()
			return mux, mux
		}),
		fx.Decorate(func(handler http.Handler) http.Handler {
			return otelhttp.NewHandler(handler, "http-server")
		}),
		fx.Invoke(func(conf server.Config, logger *zap.Logger) {
			logger.Info("server config loaded", zap.Any("config", conf))
		}),
		fx.Invoke(startHTTPServer),
	)
}

func provideConfig(loader *coreconf.Loader, logger *zap.Logger) (server.Config, error) {
	return coreconf.Load[server.Config](loader, "server", nil)
}

func startHTTPServer(lc fx.Lifecycle, log *zap.Logger, conf server.Config, handler http.Handler, readiness health.ComponentManager, shutdowner fx.Shutdowner) {
	var srv *http.Server
	markReady := readiness.AddComponent("http-server")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create server in OnStart - all routes are registered by now
			srv = server.NewServer(conf, handler)

			ln, err := net.Listen("tcp", ":"+strconv.Itoa(conf.Port))
			if err != nil {
				log.Error("failed to listen", zap.Error(err))
				return err
			}
			actualAddr := ln.Addr()
			log.Info("starting HTTP server at", zap.String("addr", actualAddr.String()))

			markReady()

			go func() {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					log.Error("HTTP server stopped with error", zap.Error(err))
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
