package server

import (
	"context"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var HttpServerModule = fx.Options(
	fx.Provide(
		ProvideNewServer,
		NewConfig,
	),
	fx.Invoke(StartHTTPServer),
)

func StartHTTPServer(Server) {}

func ProvideNewServer(lc fx.Lifecycle, log *zap.Logger, conf Config, handler http.Handler) Server {
	srv := NewServer(log, conf, handler)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srv.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
