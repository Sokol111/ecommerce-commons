package token

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// tokenOptions holds internal configuration for the token module.
type tokenOptions struct {
	config           *Config
	disable          bool
	testClaims       *Claims
	useTestValidator bool
}

// TokenOption is a functional option for configuring the token module.
type TokenOption func(*tokenOptions)

// WithTokenConfig provides a static Config (useful for tests).
func WithTokenConfig(cfg Config) TokenOption {
	return func(opts *tokenOptions) {
		opts.config = &cfg
	}
}

// WithDisableValidation disables token validation and returns noop validator.
// Useful for tests where token validation should be bypassed.
func WithDisableValidation() TokenOption {
	return func(opts *tokenOptions) {
		opts.disable = true
	}
}

// WithTestClaims provides a validator that always returns the given claims.
// Useful for tests that need specific user context.
func WithTestClaims(claims Claims) TokenOption {
	return func(opts *tokenOptions) {
		opts.testClaims = &claims
	}
}

// WithTestValidator enables the test validator that decodes base64-encoded JSON tokens.
// Use GenerateTestToken or GenerateAdminTestToken to create tokens for this validator.
// Useful for e2e/integration tests where you want realistic token flow without PASETO.
func WithTestValidator() TokenOption {
	return func(opts *tokenOptions) {
		opts.useTestValidator = true
	}
}

// NewSecurityHandlerModule provides a SecurityHandler for dependency injection.
// This is the recommended way to handle authentication in services.
// Also provides Validator for services that need direct token validation (e.g., refresh tokens).
//
// Options:
//   - WithTokenConfig: provide static token Config (useful for tests)
//   - WithDisableValidation: bypass all security, returns admin claims
//   - WithTestValidator: use base64 JSON tokens (for e2e tests with realistic flow)
//
// Example usage:
//
//	// Production - validates PASETO tokens
//	token.NewSecurityHandlerModule()
//
//	// Testing - bypass security completely
//	token.NewSecurityHandlerModule(token.WithDisableValidation())
//
//	// E2E testing - use base64 JSON tokens
//	token.NewSecurityHandlerModule(token.WithTestValidator())
func NewSecurityHandlerModule(opts ...TokenOption) fx.Option {
	cfg := &tokenOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Test validator - decodes base64 JSON tokens
	if cfg.useTestValidator {
		return fx.Provide(
			newTestValidator,
			NewSecurityHandler,
		)
	}

	// Disabled - bypass all security
	if cfg.disable {
		return fx.Provide(
			newNoopValidator,
			NewSecurityHandler,
		)
	}

	// Production - validate with real validator
	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(
			provideConfig,
			newTokenValidator,
			NewSecurityHandler,
		),
	)
}

func provideConfig(opts *tokenOptions, v *viper.Viper) (Config, error) {
	if opts.config != nil {
		return *opts.config, nil
	}
	return loadConfigFromViper(v)
}

func loadConfigFromViper(v *viper.Viper) (Config, error) {
	var cfg Config
	sub := v.Sub("security")
	if sub == nil {
		return cfg, fmt.Errorf("security configuration section is required")
	}

	tokenSub := sub.Sub("token")
	if tokenSub == nil {
		return cfg, fmt.Errorf("security.token configuration section is required")
	}

	if err := tokenSub.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load token config: %w", err)
	}

	if cfg.PublicKey == "" {
		return cfg, fmt.Errorf("security.token.public-key is required")
	}

	return cfg, nil
}
