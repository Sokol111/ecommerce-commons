package grpcclient

import (
	"errors"
	"time"
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
