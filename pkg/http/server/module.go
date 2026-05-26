package server

import (
	"context"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// serverOptions holds internal configuration for the HTTP server module.
type serverOptions struct {
	config *Config
}

// Option is a functional option for configuring the HTTP server module.
type Option func(*serverOptions)

// WithServerConfig provides a static Config (useful for tests).
func WithServerConfig(cfg Config) Option {
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
		fx.Provide(newServeMux),
		fx.Invoke(startHTTPServer),
	)
}

func provideConfig(opts *serverOptions, k *koanf.Koanf, logger *zap.Logger) (Config, error) {
	cfg, err := config.Load[Config](k, "server", opts.config)
	if err != nil {
		return Config{}, err
	}

	logger.Info("server config loaded", zap.Any("config", cfg))
	return cfg, nil
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
