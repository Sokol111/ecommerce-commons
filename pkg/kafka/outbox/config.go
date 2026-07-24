package outbox

import "time"

// Config holds the outbox pattern configuration.
type Config struct {
	// MaxBackoff is the maximum delay between retry attempts.
	// Default: 10 hours
	MaxBackoff time.Duration `koanf:"max-backoff"`
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *Config) ApplyDefaults() {
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 10 * time.Hour
	}
}

// Validate validates the outbox configuration.
func (c *Config) Validate() error {
	return nil
}
