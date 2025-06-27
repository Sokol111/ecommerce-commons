package logger

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Level string `mapstructure:"level"`
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	sub := v.Sub("logger")
	if sub == nil {
		return cfg, nil
	}

	if err := sub.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load logger config: %w", err)
	}
	return cfg, nil
}
