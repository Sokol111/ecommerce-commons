package health

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewReadinessModule() fx.Option {
	return fx.Provide(
		func(logger *zap.Logger, appConfig config.AppConfig) *readiness {
			return newReadiness(logger, appConfig.IsKubernetes)
		},
		func(r *readiness) ComponentManager { return r },
		func(r *readiness) ReadinessChecker { return r },
		func(r *readiness) ReadinessWaiter { return r },
		func(r *readiness) TrafficController { return r },
	)
}
