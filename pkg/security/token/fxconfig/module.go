package token

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
)

// clientCredentialsOptions holds internal configuration for the token module.
type clientCredentialsOptions struct {
	config *token.Config
}

// Option is a functional option for configuring the token module.
type Option func(*clientCredentialsOptions)

// WithClientCredentialsConfig provides a static Config (useful for tests).
func WithClientCredentialsConfig(cfg token.Config) Option {
	return func(opts *clientCredentialsOptions) {
		opts.config = &cfg
	}
}

// NewClientCredentialsModule provides an oauth2.TokenSource for outgoing service-to-service calls.
// Reads configuration from security.client-credentials.
// Use WithClientCredentialsConfig for static config (useful for tests).
func NewClientCredentialsModule(opts ...Option) fx.Option {
	cfg := &clientCredentialsOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.Supply(cfg),
		fx.Provide(provideConfig),
		fx.Provide(provideTokenSource),
	)
}

func provideConfig(opts *clientCredentialsOptions, loader *coreconfig.Loader) (token.Config, error) {
	return coreconfig.Load[token.Config](loader, "security.client-credentials", opts.config)
}

func provideTokenSource(cfg token.Config) (oauth2.TokenSource, error) {
	if cfg.Enabled != nil && !*cfg.Enabled {
		return nil, nil
	}
	return token.NewTokenSource(cfg)
}
