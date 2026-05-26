package logger

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// Config holds the configuration for the logger.
type Config struct {
	// Level is the log level string: "debug", "info", "warn", "error", "dpanic", "panic", "fatal".
	Level string `koanf:"level"`

	// Development enables development mode with console encoding and human-readable timestamps.
	// In production mode (false), JSON encoding is used.
	Development bool `koanf:"development"`

	// parsedLevel is the parsed zapcore.Level, populated during Validate().
	parsedLevel zapcore.Level
}

// ParsedLevel returns the parsed zapcore.Level.
// Must be called after Validate().
func (c *Config) ParsedLevel() zapcore.Level {
	return c.parsedLevel
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *Config) ApplyDefaults() {
	if c.Level == "" {
		c.Level = "info"
	}
}

// Validate validates the configuration and parses Level string.
func (c *Config) Validate() error {
	parsed, err := zapcore.ParseLevel(c.Level)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", c.Level, err)
	}
	c.parsedLevel = parsed
	return nil
}
