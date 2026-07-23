package config

import "fmt"

// Configurable is the interface that config structs must implement
// to use the Loader.
// Methods must be defined on the pointer receiver (*T).
type Configurable interface {
	ApplyDefaults()
	Validate() error
}

// Source is the minimal surface from koanf that Loader needs.
// It is defined as an unexported interface so the package can still
// test Loading without exposing koanf to consumers.
type Source interface {
	Exists(key string) bool
	Unmarshal(key string, target any) error
}

// Loader loads configuration from a source, applies defaults, and validates.
type Loader struct {
	k Source
}

// NewLoader creates a Loader backed by the given source.
func NewLoader(k Source) *Loader {
	return &Loader{k: k}
}

// Load reads configuration at the given key from the source, applies defaults, and validates.
//
// If override is non-nil, it is used instead of loading from the source.
// The order is always: override → unmarshal → apply defaults → validate.
//
// Type parameters:
//   - T: the config struct type (e.g., kafka.Config)
//   - PT: pointer to T that implements Configurable (inferred automatically)
//
// Usage:
//
//	loader := config.NewLoader(source)
//	cfg, err := config.Load[Config](loader, "kafka", opts.config)
func Load[T any, PT interface {
	*T
	Configurable
}](l *Loader, key string, override *T) (T, error) {
	var cfg T

	if override != nil {
		cfg = *override
	} else if l.k.Exists(key) {
		if err := l.k.Unmarshal(key, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to load %s config: %w", key, err)
		}
	}

	PT(&cfg).ApplyDefaults()

	if err := PT(&cfg).Validate(); err != nil {
		return cfg, fmt.Errorf("invalid %s config: %w", key, err)
	}

	return cfg, nil
}
