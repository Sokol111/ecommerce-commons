package validation

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
)

// options holds internal configuration for the jwks module.
type options struct {
	config           *Config
	disable          bool
	useTestValidator bool
}

// Option is a functional option for configuring the jwks module.
type Option func(*options)

// WithConfig provides a static Config (useful for tests).
func WithConfig(cfg Config) Option {
	return func(opts *options) {
		opts.config = &cfg
	}
}

// WithDisableValidation disables token validation and returns noop validator.
// Useful for tests where token validation should be bypassed.
func WithDisableValidation() Option {
	return func(opts *options) {
		opts.disable = true
	}
}

// WithTestValidator enables the test validator that decodes base64-encoded JSON tokens.
// Use GenerateTestToken or GenerateAdminTestToken to create tokens for this validator.
// Useful for e2e/integration tests where you want realistic token flow without JWT JWKS.
func WithTestValidator() Option {
	return func(opts *options) {
		opts.useTestValidator = true
	}
}

// NewModule provides SecurityHandler and Validator for dependency injection.
//
// Example usage:
//
//	// Production - validates JWT tokens via JWKS
//	validation.NewModule()
//
//	// Testing - bypass security completely
//	validation.NewModule(validation.WithDisableValidation())
//
//	// E2E testing - use base64 JSON tokens
//	validation.NewModule(validation.WithTestValidator())
func NewModule(opts ...Option) fx.Option {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.useTestValidator {
		return fx.Provide(
			newTestValidator,
			newSecurityHandler,
		)
	}

	if cfg.disable {
		return fx.Provide(
			newNoopValidator,
			newSecurityHandler,
		)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(
			provideConfig,
			newTokenValidator,
			newSecurityHandler,
		),
	)
}

func provideConfig(opts *options, k *koanf.Koanf) (Config, error) {
	if opts.config != nil {
		return *opts.config, nil
	}
	return loadConfig(k)
}

func loadConfig(k *koanf.Koanf) (Config, error) {
	var cfg Config
	if !k.Exists("security") {
		return cfg, fmt.Errorf("security configuration section is required")
	}

	if !k.Exists("security.jwks") {
		return cfg, fmt.Errorf("security.jwks configuration section is required")
	}

	if err := k.Unmarshal("security.jwks", &cfg); err != nil {
		return cfg, fmt.Errorf("failed to load jwks config: %w", err)
	}

	if cfg.JwksURL == "" {
		return cfg, fmt.Errorf("security.jwks.jwks-url is required")
	}

	return cfg, nil
}
