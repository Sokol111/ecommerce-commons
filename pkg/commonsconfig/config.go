package commonsconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func NewViper() (*viper.Viper, error) {
	v := viper.New()
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.yml"
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file [%s] does not exist: %w", configPath, err)
	}
	v.SetConfigFile(configPath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", configPath, err)
	}
	return v, nil
}
