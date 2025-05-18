package commonsserver

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConf struct {
	Port int `mapstructure:"port"`
}

func NewServerConfig(v *viper.Viper) (ServerConf, error) {
	var cfg ServerConf
	if err := v.Sub("server").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load server config: %w", err)
	}
	return cfg, nil
}
