package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Brokers         string          `mapstructure:"brokers"`
	ConsumersConfig ConsumersConfig `mapstructure:"consumers-config"`
}

type ConsumersConfig struct {
	GroupID         string           `mapstructure:"group-id"`
	AutoOffsetReset string           `mapstructure:"auto-offset-reset"`
	ConsumerConfig  []ConsumerConfig `mapstructure:"consumers"`
}

type ConsumerConfig struct {
	Name            string `mapstructure:"name"`
	Topic           string `mapstructure:"topic"`
	GroupID         string `mapstructure:"group-id"`
	AutoOffsetReset string `mapstructure:"auto-offset-reset"`
}

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
	logger.Info("loaded kafka config", zap.Any("config", cfg))
	return cfg, nil
}
