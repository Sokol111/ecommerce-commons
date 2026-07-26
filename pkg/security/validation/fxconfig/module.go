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

// validationOptions holds internal configuration for the validation module.
type validationOptions struct {
	config *validation.Config
}

// Option is a functional option for configuring the validation module.
type Option func(*validationOptions)

// WithJWKSConfig provides a static Config (useful for tests).
func WithJWKSConfig(cfg validation.Config) Option {
	return func(opts *validationOptions) {
		opts.config = &cfg
	}
}

// NewJWKSModule provides SecurityHandler and Validator for dependency injection.
//
// Example usage:
//
//	// Production - validates JWT tokens via JWKS
//	validation.NewJWKSModule()
func NewJWKSModule(opts ...Option) fx.Option {
	cfg := &validationOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("security-validation",
		fx.Supply(cfg),
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

func provideConfig(opts *validationOptions, loader *coreconfig.Loader) (validation.Config, error) {
	return coreconfig.Load[validation.Config](loader, "security.jwks", opts.config)
}
