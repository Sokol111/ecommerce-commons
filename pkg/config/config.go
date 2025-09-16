package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    Environment
}

type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvProduction  Environment = "prod"
)

func (e Environment) isValid() bool {
	switch e {
	case EnvDevelopment, EnvProduction:
		return true
	}
	return false
}

func NewViperModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAppConf,
			newViper,
		),
		fx.Invoke(func(logger *zap.Logger, conf Config) {
			logger.Info("Loaded environment", zap.Any("env", conf.Environment))
		}),
	)
}

func provideAppConf() (Config, error) {
	_ = godotenv.Load()
	env := Environment(os.Getenv("APP_ENV"))
	if !env.isValid() {
		return Config{}, fmt.Errorf("invalid APP_ENV: %s", env)
	}
	serviceName := os.Getenv("APP_SERVICE_NAME")
	if serviceName == "" {
		return Config{}, fmt.Errorf("APP_SERVICE_NAME is not set")
	}
	serviceVersion := os.Getenv("APP_SERVICE_VERSION")
	if serviceVersion == "" {
		return Config{}, fmt.Errorf("APP_SERVICE_VERSION is not set")
	}
	return Config{ServiceName: serviceName, ServiceVersion: serviceVersion, Environment: env}, nil
}

func newViper(conf Config) (*viper.Viper, error) {
	v := viper.New()

	configName := "config." + string(conf.Environment)
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
