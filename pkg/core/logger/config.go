package logger

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the logger configuration.
type Config struct {
	// Level specifies the minimum logging level (debug, info, warn, error, fatal, panic).
	Level string `mapstructure:"level"`

	// Development enables development mode with console encoding and human-readable timestamps.
	// In production mode (false), JSON encoding is used.
	Development bool `mapstructure:"development"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// If empty, defaults to stderr.
	OutputPaths []string `mapstructure:"outputPaths"`

	// ErrorOutputPaths is a list of URLs or file paths to write internal logger errors to.
	// If empty, defaults to stderr.
	ErrorOutputPaths []string `mapstructure:"errorOutputPaths"`
}

// Validate checks if the logger configuration is valid.
// Returns an error if any configuration parameter is invalid.
func (c Config) Validate() error {
	// Validate log level
	if c.Level != "" {
		validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
		if !contains(validLevels, c.Level) {
			return fmt.Errorf("invalid log level '%s': must be one of %v", c.Level, validLevels)
		}
	}

	// Validate output paths
	if err := validatePaths(c.OutputPaths, "outputPaths"); err != nil {
		return err
	}

	// Validate error output paths
	if err := validatePaths(c.ErrorOutputPaths, "errorOutputPaths"); err != nil {
		return err
	}

	return nil
}

// validatePaths validates that paths are not empty strings
func validatePaths(paths []string, fieldName string) error {
	for i, path := range paths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("%s[%d] cannot be empty or whitespace", fieldName, i)
		}
	}
	return nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	sub := v.Sub("logger")
	if sub == nil {
		return cfg, nil
	}

	if err := sub.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load logger config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid logger configuration: %w", err)
	}

	return cfg, nil
}
