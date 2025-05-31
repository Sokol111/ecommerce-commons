package server

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Port int `mapstructure:"port"`
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("server").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}
	return cfg, nil
}
