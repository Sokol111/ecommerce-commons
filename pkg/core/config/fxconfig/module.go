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

// configOptions holds configuration for the config module.
type configOptions struct {
	dotenvPath   string
	skipDotEnv   bool
	configPath   *string
	noConfigFile bool
	appConfig    *config.AppConfig
}

// Option is a functional option for configuring the config module.
type Option func(*configOptions)

// WithDotEnvPath sets a custom path to the .env file.
func WithDotEnvPath(path string) Option {
	return func(cfg *configOptions) {
		cfg.dotenvPath = path
	}
}

// WithoutDotEnv disables loading of a .env file.
func WithoutDotEnv() Option {
	return func(cfg *configOptions) {
		cfg.skipDotEnv = true
	}
}

// WithConfigPath sets a direct path to the configuration file.
func WithConfigPath(path string) Option {
	return func(cfg *configOptions) {
		cfg.configPath = &path
	}
}

// WithoutConfigFile disables loading of any config file.
func WithoutConfigFile() Option {
	return func(cfg *configOptions) {
		cfg.noConfigFile = true
	}
}

// WithAppConfig provides a static AppConfig (useful for tests).
func WithAppConfig(cfg config.AppConfig) Option {
	return func(opts *configOptions) {
		opts.appConfig = &cfg
	}
}

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
func NewConfigModule(opts ...Option) fx.Option {
	cfg := &configOptions{dotenvPath: ".env"}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("commons-config",
		fx.Supply(cfg),
		fx.Supply(loadDotEnv(cfg)),
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
			if !cfg.skipDotEnv {
				if dotenvLoaded {
					logger.Info("Loaded .env file", zap.String("path", cfg.dotenvPath))
				} else {
					logger.Debug("No .env file loaded", zap.String("path", cfg.dotenvPath))
				}
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

// loadDotEnv loads the .env file unless skipped. Returns true if loaded.
func loadDotEnv(cfg *configOptions) (dotenvLoaded, error) {
	if cfg.skipDotEnv {
		return dotenvLoaded(false), nil
	}

	err := godotenv.Load(cfg.dotenvPath)
	if err == nil {
		return dotenvLoaded(true), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return dotenvLoaded(false), nil
	}

	return dotenvLoaded(false), fmt.Errorf("failed to load .env file %q: %w", cfg.dotenvPath, err)
}

// loadAppConfig returns static AppConfig or loads from environment variables.
func loadAppConfig(loader *config.Loader, cfg *configOptions) (config.AppConfig, error) {
	return config.Load[config.AppConfig](loader, "", cfg.appConfig)
}

type configPath string

// resolveConfigPath resolves the config file path from environment or option.
func resolveConfigPath(cfg *configOptions) configPath {
	if cfg.noConfigFile {
		return configPath("")
	}
	if cfg.configPath != nil {
		return configPath(*cfg.configPath)
	}
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configPath(configFile)
	}
	return configPath("")
}
