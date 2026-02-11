package core

import (
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
)

// coreOptions holds internal configuration for the core module.
type coreOptions struct {
	appConfig          *config.AppConfig
	loggerConfig       *logger.Config
	disableDotEnv      bool
	disableViperConfig bool
}

// Option is a functional option for configuring the core module.
type Option func(*coreOptions)

// WithAppConfig provides a static AppConfig (useful for tests).
// When set, the AppConfig will not be loaded from environment variables.
func WithAppConfig(cfg config.AppConfig) Option {
	return func(opts *coreOptions) {
		opts.appConfig = &cfg
	}
}

// WithLoggerConfig provides a static logger Config (useful for tests).
// When set, the logger configuration will not be loaded from viper.
func WithLoggerConfig(cfg logger.Config) Option {
	return func(opts *coreOptions) {
		opts.loggerConfig = &cfg
	}
}

// WithoutEnvFile disables loading of .env file.
// Useful for tests or environments where .env files are not used.
func WithoutEnvFile() Option {
	return func(opts *coreOptions) {
		opts.disableDotEnv = true
	}
}

// WithoutConfigFile disables loading of config file (config.yaml).
// Useful for tests where configuration is provided via options.
func WithoutConfigFile() Option {
	return func(opts *coreOptions) {
		opts.disableViperConfig = true
	}
}

// NewCoreModule provides core functionality: config, logger, and health.
// It also sets increased startup and shutdown timeouts for fx application lifecycle.
//
// Options:
//   - WithAppConfig: provide static AppConfig (useful for tests)
//   - WithLoggerConfig: provide static logger Config (useful for tests)
//
// Example usage:
//
//	// Production - loads config from environment/viper
//	core.NewCoreModule()
//
//	// Testing - with static configs
//	core.NewCoreModule(
//	    core.WithAppConfig(config.AppConfig{...}),
//	    core.WithLoggerConfig(logger.Config{...}),
//	    core.WithoutEnvFile(),
//	    core.WithoutConfigFile(),
//	)
func NewCoreModule(opts ...Option) fx.Option {
	cfg := &coreOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.StartTimeout(5*time.Minute),
		fx.StopTimeout(5*time.Minute),

		dotEnvModule(cfg),
		viperModule(cfg),
		appConfigModule(cfg),
		loggerModule(cfg),
		health.NewReadinessModule(),
	)
}

func dotEnvModule(cfg *coreOptions) fx.Option {
	if cfg.disableDotEnv {
		return fx.Options()
	}
	return config.NewDotEnvModule()
}

func viperModule(cfg *coreOptions) fx.Option {
	if cfg.disableViperConfig {
		return config.NewViperModule(config.WithoutConfigFile())
	}
	return config.NewViperModule()
}

func appConfigModule(cfg *coreOptions) fx.Option {
	if cfg.appConfig != nil {
		return config.NewAppConfigModule(config.WithAppConfig(*cfg.appConfig))
	}
	return config.NewAppConfigModule()
}

func loggerModule(cfg *coreOptions) fx.Option {
	if cfg.loggerConfig != nil {
		return logger.NewZapLoggingModule(logger.WithLoggerConfig(*cfg.loggerConfig))
	}
	return logger.NewZapLoggingModule()
}
