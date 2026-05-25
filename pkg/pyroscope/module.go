// Package pyroscope provides Grafana Pyroscope continuous profiling integration.
//
// Usage:
//
//	pyroscope.NewPyroscopeModule()
//
// Configuration (koanf):
//
//	pyroscope:
//	  enabled: true
//	  endpoint: "http://pyroscope:4040"
//	  basic-auth-user: ""      # optional: Grafana Cloud instance ID
//	  basic-auth-password: ""  # optional: Grafana Cloud API key
package pyroscope

import (
	"context"
	"runtime"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/grafana/pyroscope-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewPyroscopeModule returns an fx.Option that provides continuous profiling.
// If pyroscope.enabled is false or not set, the module does nothing.
func NewPyroscopeModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
		fx.Invoke(startProfiler),
	)
}

func startProfiler(lc fx.Lifecycle, cfg Config, appCfg appconfig.AppConfig, log *zap.Logger) {
	if !cfg.Enabled {
		return
	}

	profileTypes := []pyroscope.ProfileType{
		pyroscope.ProfileCPU,
		pyroscope.ProfileInuseObjects,
		pyroscope.ProfileInuseSpace,
		pyroscope.ProfileAllocObjects,
		pyroscope.ProfileAllocSpace,
		pyroscope.ProfileGoroutines,
	}

	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	pyroCfg := pyroscope.Config{
		ApplicationName: appCfg.ServiceName,
		ServerAddress:   cfg.Endpoint,
		Logger:          &zapLogger{log: log},
		Tags: map[string]string{
			"service":     appCfg.ServiceName,
			"version":     appCfg.ServiceVersion,
			"environment": appCfg.Environment,
		},
		ProfileTypes: profileTypes,
	}

	if cfg.BasicAuthUser != "" {
		pyroCfg.BasicAuthUser = cfg.BasicAuthUser
		pyroCfg.BasicAuthPassword = cfg.BasicAuthPassword
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			profiler, err := pyroscope.Start(pyroCfg)
			if err != nil {
				log.Error("failed to start pyroscope profiler", zap.Error(err))
				return nil // non-fatal: don't prevent app from starting
			}
			log.Info("pyroscope profiling enabled",
				zap.String("endpoint", cfg.Endpoint),
				zap.String("app", appCfg.ServiceName),
			)

			lc.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					log.Info("pyroscope profiler stopping")
					return profiler.Stop()
				},
			})
			return nil
		},
	})
}

// zapLogger adapts zap.Logger to pyroscope.Logger interface.
type zapLogger struct {
	log *zap.Logger
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.log.Sugar().Infof(format, args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.log.Sugar().Debugf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.log.Sugar().Errorf(format, args...)
}
