package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func LoadConfig[T any](configFile string) (*T, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file [%s] does not exist: %w", configFile, err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
	}
	var conf T
	if err := v.UnmarshalExact(&conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file [%s] to type [%T]: %w", configFile, conf, err)
	}
	return &conf, nil
}
