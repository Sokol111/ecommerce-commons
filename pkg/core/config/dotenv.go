package config

import (
	"context"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// dotenvConfig holds configuration for the dotenv module.
type dotenvConfig struct {
	path   string
	loaded bool
}

// DotEnvOption is a functional option for configuring the dotenv module.
type DotEnvOption func(*dotenvConfig)

// WithDotEnvPath sets a custom path to the .env file.
func WithDotEnvPath(path string) DotEnvOption {
	return func(cfg *dotenvConfig) {
		cfg.path = path
	}
}

// NewDotEnvModule loads environment variables from a .env file.
// By default, loads from ".env" in the current directory.
// Loading happens synchronously when the module is created.
func NewDotEnvModule(opts ...DotEnvOption) fx.Option {
	cfg := &dotenvConfig{path: ".env"}
	for _, opt := range opts {
		opt(cfg)
	}

	err := godotenv.Load(cfg.path)
	cfg.loaded = err == nil

	return fx.Module("dotenv",
		fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					if cfg.loaded {
						logger.Info("Loaded .env file", zap.String("path", cfg.path))
					} else {
						logger.Debug("No .env file loaded", zap.String("path", cfg.path))
					}
					return nil
				},
			})
		}),
	)
}
