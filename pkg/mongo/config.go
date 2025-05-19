package mongo

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	ReplicaSet string `mapstructure:"replica-set"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Database   string `mapstructure:"database"`
}

func NewConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("mongo").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo config: %w", err)
	}
	return cfg, nil
}
