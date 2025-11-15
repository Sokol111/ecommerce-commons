package health

import (
	"go.uber.org/fx"
)

func NewHealthModule() fx.Option {
	return fx.Provide(NewReadiness)
}
