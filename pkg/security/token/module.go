package token

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
)

// tokenOptions holds internal configuration for the token module.
type tokenOptions struct {
	config *Config
}

// Option is a functional option for configuring the token module.
type Option func(*tokenOptions)

// WithConfig provides a static Config (useful for tests).
func WithConfig(cfg Config) Option {
	return func(opts *tokenOptions) {
		opts.config = &cfg
	}
}

// NewModule provides an oauth2.TokenSource for outgoing service-to-service calls.
// Reads configuration from security.client-credentials.
// Use WithConfig for static config (useful for tests).
func NewModule(opts ...Option) fx.Option {
	cfg := &tokenOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(provideConfig),
		fx.Provide(provideTokenSource),
	)
}

func provideConfig(opts *tokenOptions, k *koanf.Koanf) (Config, error) {
	return coreconfig.Load[Config](k, "security.client-credentials", opts.config)
}

func provideTokenSource(cfg Config) (oauth2.TokenSource, error) {
	return newTokenSource(cfg)
}
