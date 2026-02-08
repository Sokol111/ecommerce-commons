package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// viperConfig holds internal configuration options for the Viper module.
type viperConfig struct {
	configPath *string
}

// ViperOption is a functional option for configuring the Viper module.
type ViperOption func(*viperConfig)

// WithConfigPath sets a direct path to the configuration file.
// Overrides the default behavior of resolving from environment variables.
func WithConfigPath(path string) ViperOption {
	return func(cfg *viperConfig) {
		cfg.configPath = &path
	}
}

// NewViperModule creates an fx module for Viper configuration.
// By default, resolves config path from CONFIG_FILE or APP_ENV environment variables.
// If env vars are not set, creates an empty Viper instance.
// Use WithConfigPath to override with a direct path.
func NewViperModule(opts ...ViperOption) fx.Option {
	cfg := &viperConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("viper",
		fx.Provide(func(logger *zap.Logger) (*viper.Viper, error) {
			configFile := resolveConfigPath(cfg)
			return newViper(configFile, logger)
		}),
		fx.Invoke(func(logger *zap.Logger, v *viper.Viper) {
			logger.Info("Configuration loaded successfully",
				zap.String("configFile", v.ConfigFileUsed()),
				zap.Int("settingsCount", len(v.AllSettings())),
				zap.Strings("configKeys", v.AllKeys()),
			)
		}),
	)
}

// resolveConfigPath determines the config file path.
// If WithConfigPath was used, returns that path.
// Otherwise resolves from CONFIG_FILE or APP_ENV environment variables.
func resolveConfigPath(cfg *viperConfig) string {
	if cfg.configPath != nil {
		return *cfg.configPath
	}
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		return filepath.Join("./configs", "config."+env+".yaml")
	}
	return ""
}

func newViper(configFile string, logger *zap.Logger) (*viper.Viper, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if configFile == "" {
		logger.Info("No config file specified, using empty viper instance")
		return v, nil
	}

	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
	}

	return v, nil
}
