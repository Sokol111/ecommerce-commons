// Package profiling provides Grafana Pyroscope continuous profiling integration.
package profiling

import (
	"context"
	"runtime"

	appconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/observability/config"
	"github.com/grafana/pyroscope-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewProfilingModule returns an fx.Option that provides continuous profiling.
func NewProfilingModule() fx.Option {
	return fx.Invoke(startProfiler)
}

func startProfiler(lc fx.Lifecycle, cfg config.Config, appCfg appconfig.AppConfig, log *zap.Logger) {
	if !cfg.Profiling.Enabled {
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
		ServerAddress:   cfg.Profiling.Endpoint,
		Logger:          &zapLogger{log: log},
		Tags: map[string]string{
			"version":     appCfg.ServiceVersion,
			"environment": appCfg.Environment,
		},
		ProfileTypes: profileTypes,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			profiler, err := pyroscope.Start(pyroCfg)
			if err != nil {
				log.Error("failed to start pyroscope profiler", zap.Error(err))
				return nil // non-fatal: don't prevent app from starting
			}
			log.Info("pyroscope profiling enabled",
				zap.String("endpoint", cfg.Profiling.Endpoint),
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

func (l *zapLogger) Debugf(_ string, _ ...interface{}) {}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.log.Sugar().Errorf(format, args...)
}
