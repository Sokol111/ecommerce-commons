package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewKafkaConfigModule provides Kafka configuration for dependency injection.
func NewKafkaConfigModule() fx.Option {
	return fx.Provide(
		newConfig,
	)
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("kafka").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load kafka config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&cfg); err != nil {
		return cfg, err
	}

	// Apply default values
	applyDefaults(&cfg)

	logger.Info("loaded kafka config")
	return cfg, nil
}
