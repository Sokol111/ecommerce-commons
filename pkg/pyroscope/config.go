package pyroscope

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

// Config holds Pyroscope profiling configuration.
type Config struct {
	Enabled           bool   `koanf:"enabled"`
	Endpoint          string `koanf:"endpoint"`
	BasicAuthUser     string `koanf:"basic-auth-user"`
	BasicAuthPassword string `koanf:"basic-auth-password"`
}

func (c Config) validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required when pyroscope is enabled")
	}

	if c.BasicAuthUser != "" && c.BasicAuthPassword == "" {
		return fmt.Errorf("basic-auth-password is required when basic-auth-user is set")
	}

	if c.BasicAuthPassword != "" && c.BasicAuthUser == "" {
		return fmt.Errorf("basic-auth-user is required when basic-auth-password is set")
	}

	return nil
}

func provideConfig(k *koanf.Koanf, logger *zap.Logger) (Config, error) {
	var cfg Config
	if k.Exists("pyroscope") {
		if err := k.Unmarshal("pyroscope", &cfg); err != nil {
			return cfg, fmt.Errorf("failed to load pyroscope config: %w", err)
		}
	}

	if err := cfg.validate(); err != nil {
		return cfg, fmt.Errorf("invalid pyroscope config: %w", err)
	}

	logger.Info("loaded pyroscope config",
		zap.Bool("enabled", cfg.Enabled),
		zap.String("endpoint", cfg.Endpoint),
	)
	return cfg, nil
}
