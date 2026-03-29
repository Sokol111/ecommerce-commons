package outbox

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/v2"
)

// Config holds the outbox pattern configuration.
type Config struct {
	// MaxBackoff is the maximum delay between retry attempts.
	// Default: 10 hours
	MaxBackoff time.Duration `koanf:"max-backoff"`
}

func newConfig(k *koanf.Koanf) (*Config, error) {
	cfg := &Config{}

	if k.Exists("outbox") {
		if err := k.Unmarshal("outbox", cfg); err != nil {
			return nil, fmt.Errorf("failed to load outbox config: %w", err)
		}
	}

	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 10 * time.Hour
	}

	return cfg, nil
}
