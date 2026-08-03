package token

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/security/token"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
)

// NewClientCredentialsModule provides an oauth2.TokenSource for outgoing service-to-service calls.
// Reads configuration from security.client-credentials.
func NewClientCredentialsModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
		fx.Provide(provideTokenSource),
	)
}

func provideConfig(loader *coreconfig.Loader) (token.Config, error) {
	return coreconfig.Load[token.Config](loader, "security.client-credentials", nil)
}

func provideTokenSource(cfg token.Config) (oauth2.TokenSource, error) {
	if cfg.Enabled != nil && !*cfg.Enabled {
		//nolint:nilnil // No token source is required when client credentials are disabled (e.g. integration tests).
		return nil, nil
	}
	return token.NewTokenSource(cfg)
}
