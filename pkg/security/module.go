package security

import (
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"go.uber.org/fx"
)

// securityOptions holds internal configuration for the security module.
type securityOptions struct {
	tokenConfig *token.Config
	disable     bool
	testClaims  *token.Claims
}

// SecurityOption is a functional option for configuring the security module.
type SecurityOption func(*securityOptions)

// WithTokenConfig provides a static token Config (useful for tests).
// When set, the token configuration will not be loaded from viper.
func WithTokenConfig(cfg token.Config) SecurityOption {
	return func(opts *securityOptions) {
		opts.tokenConfig = &cfg
	}
}

// WithoutSecurity disables token validation.
// Useful for tests where security should be bypassed.
func WithoutSecurity() SecurityOption {
	return func(opts *securityOptions) {
		opts.disable = true
	}
}

// WithTestClaims provides a validator that always returns the given claims.
// Useful for tests that need specific user context (e.g., testing role-based access).
func WithTestClaims(claims token.Claims) SecurityOption {
	return func(opts *securityOptions) {
		opts.testClaims = &claims
	}
}

// NewSecurityModule provides security functionality: token validation.
//
// Options:
//   - WithTokenConfig: provide static token Config (useful for tests)
//   - WithoutSecurity: disable token validation (useful for tests)
//   - WithTestClaims: provide specific claims (useful for role-based tests)
//
// Example usage:
//
//	// Production - loads config from viper
//	security.NewSecurityModule()
//
//	// Testing - disable security (full access)
//	security.NewSecurityModule(
//	    security.WithoutSecurity(),
//	)
//
//	// Testing - with specific user claims
//	security.NewSecurityModule(
//	    security.WithTestClaims(token.Claims{
//	        UserID: "user-123",
//	        Role:   "editor",
//	        Permissions: []string{"product:read", "product:write"},
//	        Type:   "access",
//	    }),
//	)
func NewSecurityModule(opts ...SecurityOption) fx.Option {
	cfg := &securityOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		tokenModule(cfg),
	)
}

func tokenModule(cfg *securityOptions) fx.Option {
	if cfg.testClaims != nil {
		return token.NewValidatorModule(token.WithTestClaims(*cfg.testClaims))
	}
	if cfg.disable {
		return token.NewValidatorModule(token.WithDisableValidation())
	}
	if cfg.tokenConfig != nil {
		return token.NewValidatorModule(token.WithTokenConfig(*cfg.tokenConfig))
	}
	return token.NewValidatorModule()
}
