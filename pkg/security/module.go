package security

import (
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"go.uber.org/fx"
)

// NewSecurityModule provides security functionality: token validation.
func NewSecurityModule() fx.Option {
	return fx.Options(
		token.NewValidatorModule(),
	)
}
