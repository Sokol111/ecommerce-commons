package token

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
)

// NewModule provides an oauth2.TokenSource for outgoing service-to-service calls.
// Reads configuration from security.client-credentials.
func NewModule() fx.Option {
	return fx.Provide(
		provideConfig,
		provideTokenSource,
	)
}

func provideConfig(k *koanf.Koanf) (Config, error) {
	return loadConfig(k)
}

func provideTokenSource(cfg Config) (oauth2.TokenSource, error) {
	return newTokenSource(cfg)
}

func loadConfig(k *koanf.Koanf) (Config, error) {
	var cfg Config
	if !k.Exists("security.client-credentials") {
		return cfg, nil
	}
	if err := k.Unmarshal("security.client-credentials", &cfg); err != nil {
		return cfg, fmt.Errorf("failed to load client-credentials config: %w", err)
	}
	if cfg.ClientID == "" {
		return cfg, fmt.Errorf("security.client-credentials.client-id is required")
	}
	if cfg.ClientSecret == "" {
		return cfg, fmt.Errorf("security.client-credentials.client-secret is required")
	}
	if cfg.TokenURL == "" {
		return cfg, fmt.Errorf("security.client-credentials.token-url is required")
	}
	return cfg, nil
}
