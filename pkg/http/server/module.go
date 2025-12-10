package server

import (
	"context"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newConfig,
			provideGinEngine,
		),
		fx.Invoke(startHTTPServer),
	)
}

// provideGinEngine creates a new Gin engine instance.
func provideGinEngine() (*gin.Engine, http.Handler, gin.IRouter) {
	engine := gin.New(func(e *gin.Engine) {
		e.ContextWithFallback = true
	})
	return engine, engine, engine
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
