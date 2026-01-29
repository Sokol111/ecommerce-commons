package token

import (
	"go.uber.org/fx"
)

// NewValidatorModule provides a Validator for dependency injection.
func NewValidatorModule() fx.Option {
	return fx.Provide(
		newConfig,
		newTokenValidator,
	)
}
