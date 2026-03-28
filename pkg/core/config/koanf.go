package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	envprovider "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
)

// koanfOptions holds internal configuration options for the Koanf module.
type koanfOptions struct {
	configPath   *string
	noConfigFile bool
}

// KoanfOption is a functional option for configuring the Koanf module.
type KoanfOption func(*koanfOptions)

// WithKoanfConfigPath sets a direct path to the configuration file.
func WithKoanfConfigPath(path string) KoanfOption {
	return func(cfg *koanfOptions) {
		cfg.configPath = &path
	}
}

// WithoutKoanfConfigFile disables loading of any config file.
func WithoutKoanfConfigFile() KoanfOption {
	return func(cfg *koanfOptions) {
		cfg.noConfigFile = true
	}
}

// NewKoanfModule creates an fx module for Koanf configuration.
//
// Configuration is loaded in order (later overrides earlier):
//  1. YAML config file (from CONFIG_FILE env or WithKoanfConfigPath)
//  2. Environment variables
//
// Env convention: use __ (double underscore) as level delimiter, single _ as word separator.
//
//	OBSERVABILITY__OTEL_COLLECTOR_ENDPOINT → observability.otel-collector-endpoint
//	MONGO__MAX_POOL_SIZE                   → mongo.max-pool-size
//	LOGGER__LEVEL                          → logger.level
func NewKoanfModule(opts ...KoanfOption) fx.Option {
	cfg := &koanfOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Module("koanf",
		fx.Provide(func() (*koanf.Koanf, error) {
			return newKoanf(resolveKoanfConfigPath(cfg))
		}),
	)
}

func resolveKoanfConfigPath(cfg *koanfOptions) FilePath {
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

func newKoanf(configFile FilePath) (*koanf.Koanf, error) {
	k := koanf.New(".")

	// 1. Load config file (if provided).
	if configFile != "" {
		if err := k.Load(file.Provider(string(configFile)), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to read config file [%s]: %w", configFile, err)
		}
		fmt.Printf("[config] Configuration loaded from %s (keys: %d)\n", configFile, len(k.Keys()))
	} else {
		fmt.Println("[config] No config file specified, using environment variables only")
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
