package token

import (
	"go.uber.org/fx"
)

// NewTokenValidatorModule provides a TokenValidator for dependency injection.
func NewTokenValidatorModule() fx.Option {
	return fx.Provide(
		newConfig,
		newTokenValidator,
	)
}
