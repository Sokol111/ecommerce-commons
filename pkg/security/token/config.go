package token

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the configuration for PASETO token validation.
type Config struct {
	// PublicKey is the hex-encoded Ed25519 public key for verifying tokens.
	PublicKey string `mapstructure:"public-key"`
}

func newConfig(v *viper.Viper) (Config, error) {
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
