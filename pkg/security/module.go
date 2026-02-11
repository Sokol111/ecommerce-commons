package security

import (
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"go.uber.org/fx"
)

// securityOptions holds internal configuration for the security module.
type securityOptions struct {
	tokenConfig      *token.Config
	disable          bool
	useTestValidator bool
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

// WithoutSecurity disables token validation and returns admin claims.
// Useful for unit tests where security is not the focus.
func WithoutSecurity() SecurityOption {
	return func(opts *securityOptions) {
		opts.disable = true
	}
}

// WithTestValidator enables the test validator that decodes base64-encoded JSON tokens.
// Use token.GenerateTestToken or token.GenerateAdminTestToken to create tokens.
// Useful for e2e/integration tests where you want realistic token flow without PASETO.
func WithTestValidator() SecurityOption {
	return func(opts *securityOptions) {
		opts.useTestValidator = true
	}
}

// NewSecurityModule provides security functionality: SecurityHandler for token validation.
//
// Options:
//   - WithTokenConfig: provide static token Config (useful for tests)
//   - WithoutSecurity: bypass security, returns admin claims (useful for unit tests)
//   - WithTestValidator: use base64 JSON tokens (useful for e2e tests)
//
// Example usage:
//
//	// Production - validates PASETO tokens
//	security.NewSecurityModule()
//
//	// Unit tests - bypass security completely
//	security.NewSecurityModule(
//	    security.WithoutSecurity(),
//	)
//
//	// E2E tests - use base64 JSON tokens with realistic flow
//	security.NewSecurityModule(
//	    security.WithTestValidator(),
//	)
func NewSecurityModule(opts ...SecurityOption) fx.Option {
	cfg := &securityOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		securityHandlerModule(cfg),
	)
}

func securityHandlerModule(cfg *securityOptions) fx.Option {
	if cfg.useTestValidator {
		return token.NewSecurityHandlerModule(token.WithTestValidator())
	}
	if cfg.disable {
		return token.NewSecurityHandlerModule(token.WithDisableValidation())
	}
	if cfg.tokenConfig != nil {
		return token.NewSecurityHandlerModule(token.WithTokenConfig(*cfg.tokenConfig))
	}
	return token.NewSecurityHandlerModule()
}
