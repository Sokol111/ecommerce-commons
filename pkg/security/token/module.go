package token

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// tokenOptions holds internal configuration for the token module.
type tokenOptions struct {
	config     *Config
	disable    bool
	testClaims *Claims
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

// NewValidatorModule provides a Validator for dependency injection.
// By default, configuration is loaded from viper.
// Use WithTokenConfig for static config (useful for tests).
// Use WithDisableValidation to disable validation (useful for tests).
// Use WithTestClaims to provide specific claims (useful for tests).
func NewValidatorModule(opts ...TokenOption) fx.Option {
	cfg := &tokenOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	// If test claims are provided, use test claims validator
	if cfg.testClaims != nil {
		return fx.Provide(func() Validator {
			return newTestClaimsValidator(*cfg.testClaims)
		})
	}

	// If validation is disabled, provide noop validator directly
	if cfg.disable {
		return fx.Provide(newNoopValidator)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(
			provideConfig,
			newTokenValidator,
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
