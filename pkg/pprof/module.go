package pprof

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewPprofModule returns an fx.Option that conditionally starts a pprof HTTP server.
// Configuration is read from viper:
//
//	pprof:
//	  enabled: true
//	  addr: "localhost:6060"
//
// If pprof.enabled is false or not set, the module does nothing.
func NewPprofModule() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, v *viper.Viper, log *zap.Logger) {
		if !v.GetBool("pprof.enabled") {
			return
		}

		addr := v.GetString("pprof.addr")
		if addr == "" {
			addr = "localhost:6060"
		}

		srv := &http.Server{Addr: addr}

		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				log.Info("pprof enabled", zap.String("addr", addr))
				go func() {
					if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						log.Error("pprof server failed", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Info("pprof server stopping")
				return srv.Shutdown(ctx)
			},
		})
	})
}
