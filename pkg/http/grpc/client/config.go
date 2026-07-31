package client

import (
	"errors"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
)

// Config holds shared gRPC client configuration.
type Config struct {
	Address string        `koanf:"address"`
	Timeout time.Duration `koanf:"timeout"`
}

// ApplyDefaults sets sensible defaults for gRPC client configuration.
func (c *Config) ApplyDefaults() {
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
}

// Validate ensures required gRPC client configuration fields are set.
func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("grpc client config: address is required")
	}
	return nil
}

// LoadConfig reads gRPC client configuration from the loader at the provided key
// (e.g. "tenant.grpc") using the commons config loader with defaults and validation.
func LoadConfig(loader *config.Loader, key string) (Config, error) {
	cfg, err := config.Load[Config](loader, key, nil)
	if err != nil {
		return Config{}, fmt.Errorf("load %s config: %w", key, err)
	}
	return cfg, nil
}
