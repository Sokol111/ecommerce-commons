package config

import (
	"fmt"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Environment variable names.
const (
	envAppEnv                = "APP_ENV"
	envAppServiceName        = "APP_SERVICE_NAME"
	envAppServiceVersion     = "APP_SERVICE_VERSION"
	envKubernetesServiceHost = "KUBERNETES_SERVICE_HOST"
)

// AppConfig represents the core application metadata.
type AppConfig struct {
	ServiceName    string
	ServiceVersion string
	// Environment is the deployment environment (e.g., "local", "staging", "pro")
	Environment  string
	IsKubernetes bool
}

// appConfigOptions holds internal configuration for the AppConfig module.
type appConfigOptions struct {
	config *AppConfig
}

// AppConfigOption is a functional option for configuring the AppConfig module.
type AppConfigOption func(*appConfigOptions)

// WithAppConfig provides a static AppConfig (useful for tests).
func WithAppConfig(cfg AppConfig) AppConfigOption {
	return func(opts *appConfigOptions) {
		opts.config = &cfg
	}
}

// NewAppConfigModule creates an fx module for application configuration.
// By default, loads from environment variables (APP_ENV, APP_SERVICE_NAME, APP_SERVICE_VERSION).
// Use WithAppConfig for static config (useful for tests).
func NewAppConfigModule(opts ...AppConfigOption) fx.Option {
	cfg := &appConfigOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("appconfig",
		fx.Provide(func() (AppConfig, error) {
			if cfg.config != nil {
				return *cfg.config, nil
			}
			return loadAppConfigFromEnv()
		}),
		fx.Invoke(func(logger *zap.Logger, conf AppConfig) {
			logger.Info("Loaded application configuration",
				zap.String("service", conf.ServiceName),
				zap.String("version", conf.ServiceVersion),
				zap.String("environment", conf.Environment),
				zap.Bool("isKubernetes", conf.IsKubernetes),
			)
		}),
	)
}

// loadAppConfigFromEnv creates AppConfig from environment variables.
func loadAppConfigFromEnv() (AppConfig, error) {
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

	return AppConfig{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    env,
		IsKubernetes:   os.Getenv(envKubernetesServiceHost) != "",
	}, nil
}
