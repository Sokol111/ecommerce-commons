package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Environment variable names
const (
	envAppEnv            = "APP_ENV"
	envAppServiceName    = "APP_SERVICE_NAME"
	envAppServiceVersion = "APP_SERVICE_VERSION"
	envConfigFile        = "CONFIG_FILE"
	envConfigDir         = "CONFIG_DIR"
	envConfigName        = "CONFIG_NAME"
)

// Default configuration values
const (
	defaultConfigDir = "./configs"
)

// AppConfig represents the core application metadata and configuration paths.
// This is loaded from environment variables and provides service identity
// and configuration file location information.
type AppConfig struct {
	// ConfigFile is the full path to the config file
	ConfigFile string
	// ServiceName is the name of the service
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment is the deployment environment (e.g., "local", "staging", "pro")
	Environment string
}

// NewAppConfigModule creates a new fx module for application configuration.
// It provides AppConfig instance loaded from environment variables.
//
// Required environment variables:
//   - APP_ENV: Environment name (e.g., "local", "staging", "pro")
//   - APP_SERVICE_NAME: Service name
//   - APP_SERVICE_VERSION: Service version
//
// Optional environment variables:
//   - CONFIG_FILE: Full path to config file (default: ./configs/config.{env}.yaml)
func NewAppConfigModule() fx.Option {
	return fx.Module("appconfig",
		fx.Provide(newAppConfig),
		fx.Invoke(func(logger *zap.Logger, conf AppConfig) {
			_, dotEnvExists := os.Stat(".env")
			logger.Info("Loaded application configuration",
				zap.String("service", conf.ServiceName),
				zap.String("version", conf.ServiceVersion),
				zap.String("environment", conf.Environment),
				zap.String("configFile", conf.ConfigFile),
				zap.Bool("configFileProvided", os.Getenv(envConfigFile) != ""),
				zap.Bool("dotEnvFound", dotEnvExists == nil),
			)
		}),
	)
}

// newAppConfig creates a new AppConfig by reading environment variables.
// It loads the .env file if it exists (optional).
func newAppConfig() (AppConfig, error) {
	// Load .env file if exists - silently ignore if file doesn't exist
	_ = godotenv.Load()

	env := os.Getenv(envAppEnv)
	if env == "" {
		return AppConfig{}, fmt.Errorf("%s is required", envAppEnv)
	}

	serviceName := os.Getenv(envAppServiceName)
	if serviceName == "" {
		return AppConfig{}, fmt.Errorf("%s is required", envAppServiceName)
	}

	serviceVersion := os.Getenv(envAppServiceVersion)
	if serviceVersion == "" {
		return AppConfig{}, fmt.Errorf("%s is required", envAppServiceVersion)
	}

	// Build full config file path
	configFile := os.Getenv(envConfigFile)
	if configFile == "" {
		// Get config directory from environment or use default
		configDir := os.Getenv(envConfigDir)
		if configDir == "" {
			configDir = defaultConfigDir
		}

		// Get config name from environment or build from environment type
		configName := os.Getenv(envConfigName)
		if configName == "" {
			configName = "config." + env
		}

		configFile = filepath.Join(configDir, configName+".yaml")
	}

	return AppConfig{
		ConfigFile:     configFile,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    env,
	}, nil
}
