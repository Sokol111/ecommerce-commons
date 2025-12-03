package modules

import (
	"github.com/Sokol111/ecommerce-commons/pkg/observability"
	"go.uber.org/fx"
)

// NewObservabilityModule provides observability functionality: tracing, metrics
func NewObservabilityModule() fx.Option {
	return observability.NewObservabilityModule()
}
