package mongo

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ConnectionString string `mapstructure:"connection-string"`
	Host             string `mapstructure:"host"`
	Port             int    `mapstructure:"port"`
	ReplicaSet       string `mapstructure:"replica-set"`
	Username         string `mapstructure:"username"`
	Password         string `mapstructure:"password"`
	Database         string `mapstructure:"database"`
	DirectConnection bool   `mapstructure:"direct-connection"`
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("mongo").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo config: %w", err)
	}
	return cfg, nil
}
