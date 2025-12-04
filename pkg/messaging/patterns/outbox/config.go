package outbox

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// MaxBackoff is the maximum delay between retry attempts.
	// Default: 16 minutes
	MaxBackoff time.Duration `mapstructure:"max-backoff"`
}

func newConfig(v *viper.Viper) (*Config, error) {
	cfg := &Config{}

	if sub := v.Sub("outbox"); sub != nil {
		if err := sub.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("failed to load outbox config: %w", err)
		}
	}

	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 16 * time.Minute
	}

	return cfg, nil
}
