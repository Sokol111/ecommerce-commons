package validation

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	fx_interceptor "github.com/Sokol111/ecommerce-commons/pkg/http/connect/interceptor/fxconfig"
	"github.com/Sokol111/ecommerce-commons/pkg/security/validation"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AuthInterceptorPriority is the recommended priority for the auth interceptor.
// It must run AFTER tenant resolution (18) but BEFORE tenant validation (26)
// so that claims are available for tenant validation.
const AuthInterceptorPriority = 22

// NewJWKSModule provides SecurityHandler and Validator for dependency injection.
//
// Example usage:
//
//	// Production - validates JWT tokens via JWKS
//	validation.NewJWKSModule()
func NewJWKSModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideConfig,
			provideTokenValidator,
			fx.Annotate(
				func(
					cfg validation.Config,
					validator validation.Validator,
					perms validation.ProcedurePermissions,
					log *zap.Logger,
				) fx_interceptor.Interceptor {
					if cfg.Enabled != nil && !*cfg.Enabled {
						return fx_interceptor.Interceptor{}
					}
					return fx_interceptor.Interceptor{
						Priority: AuthInterceptorPriority,
						Handler:  validation.NewAuthInterceptor(validator, perms, log),
					}
				},
				fx.ResultTags(`group:"connect_interceptor"`),
			)),
	)
}

func provideTokenValidator(cfg validation.Config) (validation.Validator, error) {
	if cfg.Enabled != nil && !*cfg.Enabled {
		return nil, nil
	}
	return validation.NewTokenValidator(cfg)
}

func provideConfig(loader *coreconfig.Loader) (validation.Config, error) {
	return coreconfig.Load[validation.Config](loader, "security.jwks", nil)
}
