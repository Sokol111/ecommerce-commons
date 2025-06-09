package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func (e Environment) isValid() bool {
	switch e {
	case EnvDevelopment, EnvProduction:
		return true
	}
	return false
}

func NewViperModule() fx.Option {
	return fx.Provide(
		provideEnv,
		newViper,
	)
}

func provideEnv() (Environment, error) {
	_ = godotenv.Load()
	env := Environment(os.Getenv("APP_ENV"))
	if !env.isValid() {
		return "", fmt.Errorf("invalid APP_ENV: %s", env)
	}
	return Environment(env), nil
}

func newViper(env Environment) (*viper.Viper, error) {
	v := viper.New()

	configName := "config." + string(env)
	configPath := "./configs"

	v.SetConfigName(configName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	fullConfigPath := fmt.Sprintf("%s/%s.yaml", configPath, configName)
	if _, err := os.Stat(fullConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file [%s] does not exist: %w", fullConfigPath, err)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", fullConfigPath, err)
	}
	return v, nil
}
