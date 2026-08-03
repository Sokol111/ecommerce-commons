package config

import (
	"errors"
)

// AppConfig represents the core application metadata.
type AppConfig struct {
	ServiceName           string `koanf:"app-service-name"`
	ServiceVersion        string `koanf:"app-service-version"`
	Environment           string `koanf:"app-env"`
	kubernetesServiceHost string `koanf:"kubernetes-service-host"`
	IsKubernetes          bool
}

// ApplyDefaults sets default values for AppConfig fields based on environment variables.
func (c *AppConfig) ApplyDefaults() {
	if c.kubernetesServiceHost != "" {
		c.IsKubernetes = true
	}
}

// Validate checks that all required fields are present.
func (c *AppConfig) Validate() error {
	var errs []error
	if c.ServiceName == "" {
		errs = append(errs, errors.New("app-service-name is required"))
	}
	if c.ServiceVersion == "" {
		errs = append(errs, errors.New("app-service-version is required"))
	}
	if c.Environment == "" {
		errs = append(errs, errors.New("app-env is required"))
	}
	return errors.Join(errs...)
}
