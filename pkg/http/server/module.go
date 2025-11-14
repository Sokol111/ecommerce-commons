package server

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHttpServerModule() fx.Option {
	return fx.Options(
		fx.Provide(newConfig),
		fx.Invoke(startHTTPServer),
	)
}

func startHTTPServer(lc fx.Lifecycle, log *zap.Logger, conf Config, engine *gin.Engine, readiness health.Readiness, shutdowner fx.Shutdowner) {
	var srv Server
	readiness.AddOne()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create server in OnStart - all routes are registered by now
			srv = newServer(log, conf, engine)
			go func() {
				if err := srv.Serve(); err != nil {
					log.Error("HTTP server failed, shutting down application", zap.Error(err))
					shutdowner.Shutdown()
				}
			}()
			readiness.Done()
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
