package grpcclient

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
)

// LoadConfig reads gRPC client configuration from the loader at the provided key
// (e.g. "tenant.grpc") using the commons config loader with defaults and validation.
func LoadConfig(loader *config.Loader, key string) (Config, error) {
	cfg, err := config.Load[Config](loader, key, nil)
	if err != nil {
		return Config{}, fmt.Errorf("load %s config: %w", key, err)
	}
	return cfg, nil
}
