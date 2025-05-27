package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Brokers   string           `mapstructure:"brokers"`
	Consumers []ConsumerConfig `mapstructure:"consumers"`
}

type ConsumerConfig struct {
	Handler         string `mapstructure:"handler"`
	Topic           string `mapstructure:"topic"`
	GroupID         string `mapstructure:"group-id"`
	AutoOffsetReset string `mapstructure:"auto-offset-reset"`
}

func NewConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("kafka").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo config: %w", err)
	}
	return cfg, nil
}
