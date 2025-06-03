package server

import (
	"context"
	"net/http"

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

func provideServer(lc fx.Lifecycle, log *zap.Logger, conf Config, handler http.Handler) Server {
	srv := newServer(log, conf, handler)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return srv.Serve()
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
