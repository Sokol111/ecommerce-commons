package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
)

// NewCoreModule provides core functionality: config and logger
func NewCoreModule() fx.Option {
	return fx.Options(
		logger.NewZapLoggingModule(),
		config.NewViperModule(),
	)
}
