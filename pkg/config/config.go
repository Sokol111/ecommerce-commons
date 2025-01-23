package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

func LoadConfig[T any](configFile string) (*T, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
	}
	var conf T
	if err := v.Unmarshal(&conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file [%s] to type [%T]: %w", configFile, conf, err)
	}
	return &conf, nil
}
