package fxconfig

import (
	"context"
	"time"

	config_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/config/fxconfig"
	health_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/health/fxconfig"
	logger_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/logger/fxconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewCoreModule provides core functionality: config, logger, and health.
// It also sets increased startup and shutdown timeouts for fx application lifecycle.
func NewCoreModule() fx.Option {
	return fx.Options(
		fx.StartTimeout(5*time.Minute),
		fx.StopTimeout(5*time.Minute),
		config_fx_pkg.NewConfigModule(),
		logger_fx_pkg.NewZapLoggingModule(),
		health_fx_pkg.NewReadinessModule(),
		fx.Invoke(func(lc fx.Lifecycle, log *zap.Logger) {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					log.Info("Application stopping...")
					return nil
				},
			})
		}),
	)
}
