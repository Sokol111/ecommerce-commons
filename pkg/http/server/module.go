package server

import (
	"context"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/http/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHttpServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideServer,
			newConfig,
		),
		fx.Invoke(startHTTPServer),
	)
}

func startHTTPServer(Server) {}

func provideServer(lc fx.Lifecycle, log *zap.Logger, conf Config, handler http.Handler, readiness health.Readiness, shutdowner fx.Shutdowner) Server {
	srv := newServer(log, conf, handler)
	readiness.AddOne()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
