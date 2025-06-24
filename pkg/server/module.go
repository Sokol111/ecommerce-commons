package server

import (
	"context"
	"net/http"

	"github.com/Sokol111/ecommerce-commons/pkg/health"
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

func provideServer(lc fx.Lifecycle, log *zap.Logger, conf Config, handler http.Handler, readiness health.Readiness) Server {
	srv := newServer(log, conf, handler)
	readiness.AddOne()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := srv.Serve()
			if err != nil {
				return err
			}
			readiness.Done()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
