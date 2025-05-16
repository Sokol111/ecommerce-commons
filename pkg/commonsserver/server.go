package commonsserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServerConf struct {
	Port int `mapstructure:"port"`
}

var HttpServerModule = fx.Options(
	fx.Provide(
		NewServer,
		NewServerConfig,
	),
	fx.Invoke(StartHTTPServer),
)

func NewServerConfig(v *viper.Viper) (ServerConf, error) {
	var cfg ServerConf
	if err := v.Sub("server").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}
	return cfg, nil
}

func StartHTTPServer(*http.Server) {}

func NewServer(lc fx.Lifecycle, log *zap.Logger, conf ServerConf, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(conf.Port),
		Handler: handler,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				log.Error("failed to listen", zap.Error(err))
				return err
			}
			log.Info("starting HTTP server at", zap.String("addr", srv.Addr))
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
