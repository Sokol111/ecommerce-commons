package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// serverOptions holds internal configuration for the HTTP server module.
type serverOptions struct {
	config *Config
}

// ServerOption is a functional option for configuring the HTTP server module.
type ServerOption func(*serverOptions)

// WithServerConfig provides a static Config (useful for tests).
func WithServerConfig(cfg Config) ServerOption {
	return func(opts *serverOptions) {
		opts.config = &cfg
	}
}

// NewHTTPServerModule provides HTTP server components for dependency injection.
// By default, configuration is loaded from viper.
// Use WithServerConfig for static config (useful for tests).
func NewHTTPServerModule(opts ...ServerOption) fx.Option {
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

func provideConfig(opts *serverOptions, v *viper.Viper, logger *zap.Logger) (Config, error) {
	var result Config
	if opts.config != nil {
		result = *opts.config
	} else {
		var err error
		result, err = loadConfigFromViper(v)
		if err != nil {
			return Config{}, err
		}
	}

	if err := result.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid server config: %w", err)
	}
	result.setDefaults()

	logger.Info("server config loaded", zap.Any("config", result))
	return result, nil
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
