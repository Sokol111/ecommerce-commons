package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewObservabilityConfigModule provides observability configuration.
func NewObservabilityConfigModule() fx.Option {
	return fx.Provide(newConfig)
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	sub := v.Sub("observability")
	if sub == nil {
		return cfg, nil
	}
	if err := sub.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load observability config: %w", err)
	}

	applyDefaults(&cfg)

	logger.Info("loaded observability config")
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Metrics.Interval == 0 {
		cfg.Metrics.Interval = DefaultMetricsInterval
	}
	if cfg.Tracing.SampleRatio == 0 {
		cfg.Tracing.SampleRatio = DefaultSampleRatio
	}
}
