package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// viperConfig holds internal configuration options for the Viper module.
type viperConfig struct {
	configPath   *string
	noConfigFile bool
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

// WithoutConfigFile disables loading of any config file.
// Viper will still be available for DI but with no file-based configuration.
func WithoutConfigFile() ViperOption {
	return func(cfg *viperConfig) {
		cfg.noConfigFile = true
	}
}

// FilePath represents the path to a configuration file.
// Empty string means no config file will be loaded.
type FilePath string

// NewViperModule creates an fx module for Viper configuration.
// By default, resolves config path from CONFIG_FILE environment variable.
// If env var is not set, creates an empty Viper instance.
// Use WithConfigPath to override with a direct path.
// Use WithoutConfigFile to disable config file loading.
func NewViperModule(opts ...ViperOption) fx.Option {
	cfg := &viperConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("viper",
		fx.Supply(resolveConfigPath(cfg)),
		fx.Provide(newViper),
		fx.Invoke(logViperConfig),
	)
}

func logViperConfig(logger *zap.Logger, v *viper.Viper) {
	logger.Info("Configuration loaded successfully",
		zap.String("configFile", v.ConfigFileUsed()),
		zap.Int("settingsCount", len(v.AllSettings())),
		zap.Strings("configKeys", v.AllKeys()),
	)
}

// resolveConfigPath determines the config file path.
// If noConfigFile is set, returns empty string.
// If WithConfigPath was used, returns that path.
// Otherwise resolves from CONFIG_FILE environment variable.
func resolveConfigPath(cfg *viperConfig) FilePath {
	if cfg.noConfigFile {
		return ""
	}
	if cfg.configPath != nil {
		return FilePath(*cfg.configPath)
	}
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return FilePath(configFile)
	}
	return ""
}

func newViper(configFile FilePath, logger *zap.Logger) (*viper.Viper, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if configFile == "" {
		logger.Info("No config file specified, using empty viper instance")
		return v, nil
	}

	v.SetConfigFile(string(configFile))
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
	}

	return v, nil
}
