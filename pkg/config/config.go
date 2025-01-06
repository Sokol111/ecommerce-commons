package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func LoadConfig[T any](configFile string) *T {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.AutomaticEnv()
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config file [%s]: %s", configFile, err))
	}
	var conf T
	err = v.Unmarshal(&conf)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config file [%s] to type [%T]: %s", configFile, conf, err))
	}
	return &conf
}
