// Package fxconfig provides an fx module for configuration loading.
package fxconfig

import (
	"errors"
	"fmt"
	"os"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type dotenvLoaded bool

// NewConfigModule creates a single fx module that wires all configuration loading:
//
//  1. .env file (unless WithoutDotEnv)
//  2. Koanf from YAML + env vars (unless WithoutConfigFile)
//  3. AppConfig from env vars (or static via WithAppConfig)
//
// Configuration is loaded in order (later overrides earlier):
//   - YAML config file
//   - Environment variables (override YAML)
//
// Env convention: use __ (double underscore) as level delimiter, single _ as word separator.
//
//	OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT → observability.otel-collector-endpoint
//	MONGO__MAX_POOL_SIZE                   → mongo.max-pool-size
//	LOGGER__LEVEL                          → logger.level
func NewConfigModule() fx.Option {
	return fx.Options(
		fx.Provide(func() (dotenvLoaded, error) {
			return loadDotEnv()
		}),
		fx.Provide(resolveConfigPath),
		fx.Provide(func(configPath configPath) (*koanf.Koanf, config.Source, error) {
			k, err := config.NewKoanf(string(configPath))
			if err != nil {
				return nil, nil, err
			}
			return k, k, nil
		}),
		fx.Provide(config.NewLoader),
		fx.Provide(loadAppConfig),
		fx.Invoke(func(logger *zap.Logger, dotenvLoaded dotenvLoaded, appCfg config.AppConfig) {
			if dotenvLoaded {
				logger.Info("Loaded .env file")
			} else {
				logger.Info("No .env file loaded")
			}

			logger.Info("Loaded application configuration",
				zap.String("service", appCfg.ServiceName),
				zap.String("version", appCfg.ServiceVersion),
				zap.String("environment", appCfg.Environment),
				zap.Bool("isKubernetes", appCfg.IsKubernetes),
			)
		}),
	)
}

// loadDotEnv loads the .env file. Returns true if loaded.
func loadDotEnv() (dotenvLoaded, error) {
	err := godotenv.Load(".env")
	if err == nil {
		return dotenvLoaded(true), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return dotenvLoaded(false), nil
	}

	return dotenvLoaded(false), fmt.Errorf("failed to load .env file: %w", err)
}

// loadAppConfig returns static AppConfig or loads from environment variables.
func loadAppConfig(loader *config.Loader) (config.AppConfig, error) {
	return config.Load[config.AppConfig](loader, "", nil)
}

type configPath string

// resolveConfigPath resolves the config file path from environment.
func resolveConfigPath() (configPath, error) {
	path := os.Getenv("CONFIG_FILE")

	if path == "" {
		return "", fmt.Errorf("CONFIG_FILE environment variable is not set")
	}
	return configPath(path), nil
}
