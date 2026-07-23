package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	envprovider "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// NewKoanf creates a koanf instance loaded from the given config file and environment variables.
//
// Configuration is loaded in order (later overrides earlier):
//  1. YAML config file (if configFile is non-empty)
//  2. Environment variables
//
// Env convention: use __ (double underscore) as level delimiter, single _ as word separator.
//
//	OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT → observability.otel-collector-endpoint
//	MONGO__MAX_POOL_SIZE                   → mongo.max-pool-size
//	LOGGER__LEVEL                          → logger.level
func NewKoanf(configFile string) (*koanf.Koanf, error) {
	k := koanf.New(".")

	// 1. Load config file (if provided).
	if configFile != "" {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
		}
	}

	// 2. Load environment variables (overrides config file).
	// Convention:
	//   __ (double underscore) = level delimiter (becomes ".")
	//   _  (single underscore) = word separator  (becomes "-")
	//
	// Examples:
	//   OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT → observability.otel-collector-endpoint
	//   MONGO__CONNECTION_STRING               → mongo.connection-string
	//   LOGGER__LEVEL                          → logger.level
	//   APP_ENV                                → app-env
	if err := k.Load(envprovider.Provider(".", envprovider.Opt{
		TransformFunc: transformEnvKey,
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return k, nil
}

// transformEnvKey converts environment variable names to koanf key paths.
//
//	__ → . (level delimiter)
//	_  → - (word separator)
//
// All keys are lowercased.
func transformEnvKey(key, value string) (string, any) {
	key = strings.ToLower(key)
	// Replace __ first (level delimiter), then _ (word separator).
	// Use a placeholder to avoid double replacement.
	key = strings.ReplaceAll(key, "__", "\x00")
	key = strings.ReplaceAll(key, "_", "-")
	key = strings.ReplaceAll(key, "\x00", ".")
	return key, value
}
