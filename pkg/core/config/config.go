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

// Config represents the core application configuration.
type Config struct {
	ConfigFile     string
	ConfigDir      string
	ConfigName     string
	ServiceName    string
	ServiceVersion string
	Environment    Environment
}

// Environment represents the deployment environment.
type Environment string

const (
	EnvStandalone  Environment = "standalone"
	EnvDevelopment Environment = "dev"
	EnvProduction  Environment = "pro"
)

// IsValid checks if the environment value is valid.
// Returns true for standalone, dev, or pro.
func (e Environment) IsValid() bool {
	switch e {
	case EnvStandalone, EnvDevelopment, EnvProduction:
		return true
	}
	return false
}

// String returns the string representation of the environment.
func (e Environment) String() string {
	return string(e)
}

// NewViperModule creates a new fx module for configuration management.
// It provides Config and *viper.Viper instances for dependency injection.
//
// The module uses the following environment variables:
//   - APP_ENV: Environment type (standalone, dev, pro) - REQUIRED
//   - APP_SERVICE_NAME: Service name - REQUIRED
//   - APP_SERVICE_VERSION: Service version - REQUIRED
//   - CONFIG_FILE: Explicit path to config file - OPTIONAL
//   - CONFIG_DIR: Directory containing config files - OPTIONAL (default: ./configs)
//   - CONFIG_NAME: Config file name without extension - OPTIONAL (default: config.{env})
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
	// Load .env file if exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		// Only log warning, don't fail - .env is optional
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to load .env file: %v\n", err)
		}
	}

	env := Environment(os.Getenv("APP_ENV"))
	if !env.IsValid() {
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

	configFile := os.Getenv("CONFIG_FILE")

	// Get config directory from environment or use default
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = "./configs"
	}

	// Get config name from environment or build from environment type
	configName := os.Getenv("CONFIG_NAME")
	if configName == "" {
		configName = "config." + string(env)
	}

	return Config{
		ConfigFile:     configFile,
		ConfigDir:      configDir,
		ConfigName:     configName,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    env,
	}, nil
}

func newViper(conf Config) (*viper.Viper, error) {
	v := viper.New()

	var fullConfigPath string

	if conf.ConfigFile == "" {
		v.SetConfigName(conf.ConfigName)
		v.SetConfigType("yaml")
		v.AddConfigPath(conf.ConfigDir)

		fullConfigPath = fmt.Sprintf("%s/%s.yaml", conf.ConfigDir, conf.ConfigName)
	} else {
		v.SetConfigFile(conf.ConfigFile)
		fullConfigPath = conf.ConfigFile
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if _, err := os.Stat(fullConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file [%s] does not exist: %w", fullConfigPath, err)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", fullConfigPath, err)
	}
	return v, nil
}
