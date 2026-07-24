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

// serverOptions holds internal configuration for the HTTP server module.
type serverOptions struct {
	config *server.Config
}

// Option is a functional option for configuring the HTTP server module.
type Option func(*serverOptions)

// WithServerConfig provides a static Config (useful for tests).
func WithServerConfig(cfg server.Config) Option {
	return func(opts *serverOptions) {
		opts.config = &cfg
	}
}

// NewHTTPServerModule provides HTTP server components for dependency injection.
// By default, configuration is loaded from koanf.
// Use WithServerConfig for static config (useful for tests).
func NewHTTPServerModule(opts ...Option) fx.Option {
	cfg := &serverOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("http-server",
		fx.Supply(cfg),
		fx.Provide(provideConfig),
		fx.Provide(func(opts *serverOptions) (*http.ServeMux, http.Handler) {
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

func provideConfig(opts *serverOptions, loader *coreconf.Loader, logger *zap.Logger) (server.Config, error) {
	return coreconf.Load[server.Config](loader, "server", opts.config)
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
