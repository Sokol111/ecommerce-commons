package fxconfig

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewReadinessModule provides readiness management components for dependency injection.
func NewReadinessModule() fx.Option {
	return fx.Provide(
		func(logger *zap.Logger, appConfig config.AppConfig) (health.ComponentManager, health.ReadinessChecker, health.ReadinessWaiter, health.TrafficController) {
			r := health.NewReadiness(logger, appConfig.IsKubernetes)
			return r, r, r, r
		},
	)
}
