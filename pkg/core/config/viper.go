package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewViperModule() fx.Option {
	return fx.Module("viper",
		fx.Provide(newViper),
		fx.Invoke(func(logger *zap.Logger, v *viper.Viper) {
			logger.Info("Configuration loaded successfully",
				zap.String("configFile", v.ConfigFileUsed()),
				zap.Int("settingsCount", len(v.AllSettings())),
				zap.Strings("configKeys", v.AllKeys()),
			)
		}),
	)
}

// newViper creates a new Viper instance configured to read from the
// configuration file specified in AppConfig.
//
// The instance is configured to:
//   - Read from config files (YAML, JSON, TOML, etc. based on file extension)
//   - Support environment variable overrides (automatically binds all env vars)
//   - Replace dots and dashes with underscores in env var names (e.g., app.name → APP_NAME)
//
// Environment variables take precedence over config file values.
// Example: SERVER_PORT env var overrides server.port from config file.
func newViper(conf AppConfig) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigFile(conf.ConfigFile)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file [%s]: %w", conf.ConfigFile, err)
	}

	return v, nil
}
