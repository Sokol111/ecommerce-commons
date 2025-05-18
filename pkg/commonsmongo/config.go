package commonsmongo

import (
	"fmt"

	"github.com/spf13/viper"
)

type MongoConf struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	ReplicaSet string `mapstructure:"replica-set"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Database   string `mapstructure:"database"`
}

func NewMongoConfig(v *viper.Viper) (MongoConf, error) {
	var cfg MongoConf
	if err := v.Sub("mongo").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load mongo config: %w", err)
	}
	return cfg, nil
}
