package grpcclient

import (
	"fmt"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/knadh/koanf/v2"
)

// LoadConfig reads gRPC client configuration from koanf at the provided key
// (e.g. "tenant.grpc") using the commons config loader with defaults and validation.
func LoadConfig(k *koanf.Koanf, key string) (Config, error) {
	cfg, err := config.Load[Config](k, key, nil)
	if err != nil {
		return Config{}, fmt.Errorf("load %s config: %w", key, err)
	}
	return cfg, nil
}
