package server

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(newConfig),
		fx.Invoke(startHTTPServer),
	)
}

func startHTTPServer(lc fx.Lifecycle, log *zap.Logger, conf Config, engine *gin.Engine, readiness health.ComponentManager, shutdowner fx.Shutdowner) {
	var srv Server
	readiness.AddComponent("http-server")
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create server in OnStart - all routes are registered by now
			srv = newServer(log, conf, engine)

			go func() {
				if err := srv.ServeWithReadyCallback(func() {
					readiness.MarkReady("http-server")
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
