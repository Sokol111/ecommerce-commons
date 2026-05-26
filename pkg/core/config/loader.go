package config

import (
	"fmt"

	"github.com/knadh/koanf/v2"
)

// Configurable is the interface that config structs must implement
// to use the generic Load function.
// Methods must be defined on the pointer receiver (*T).
type Configurable interface {
	ApplyDefaults()
	Validate() error
}

// Load unmarshals configuration from koanf, applies defaults, and validates.
//
// If override is non-nil, it is used instead of loading from koanf.
// The order is always: Load → ApplyDefaults → Validate.
//
// Type parameters:
//   - T: the config struct type (e.g., kafka.Config)
//   - PT: pointer to T that implements Configurable (inferred automatically)
//
// Usage:
//
//	cfg, err := config.Load[kafka.Config](k, "kafka", opts.config)
func Load[T any, PT interface {
	*T
	Configurable
}](k *koanf.Koanf, key string, override *T) (T, error) {
	var cfg T

	if override != nil {
		cfg = *override
	} else if k.Exists(key) {
		if err := k.Unmarshal(key, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to load %s config: %w", key, err)
		}
	}

	PT(&cfg).ApplyDefaults()

	if err := PT(&cfg).Validate(); err != nil {
		return cfg, fmt.Errorf("invalid %s config: %w", key, err)
	}

	return cfg, nil
}
