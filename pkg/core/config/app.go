package config

import (
	"fmt"
	"os"
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

// LoadAppConfigFromEnv creates AppConfig from environment variables.
func LoadAppConfigFromEnv() (AppConfig, error) {
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
