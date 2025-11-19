package modules

import (
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
)

// NewCoreModule provides core functionality: config, logger, and health
// It also sets increased startup and shutdown timeouts for fx application lifecycle
func NewCoreModule() fx.Option {
	return fx.Options(
		// Increase timeouts for startup and shutdown
		// Default is 15s which may not be enough
		fx.StartTimeout(5*time.Minute),
		fx.StopTimeout(5*time.Minute),

		config.NewAppConfigModule(),
		config.NewViperModule(),
		logger.NewZapLoggingModule(),
		health.NewHealthModule(),
	)
}
