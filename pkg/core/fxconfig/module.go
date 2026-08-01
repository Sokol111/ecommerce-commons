package fxconfig

import (
	"context"
	"time"

	config_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/config/fxconfig"
	health_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/health/fxconfig"
	logger_fx_pkg "github.com/Sokol111/ecommerce-commons/pkg/core/logger/fxconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// coreOptions holds internal configuration for the core module.
type coreOptions struct {
	configOpts []config_fx_pkg.Option
	loggerOpts []logger_fx_pkg.Option
}

// Option is a functional option for configuring the core module.
type Option func(*coreOptions)

// WithConfigOptions passes config options directly to the underlying config module.
// Use this to configure config file path, .env loading, static AppConfig, etc.
//
// Example usage:
//
//	// Testing - with static configs
//	core.NewCoreModule(
//	    core.WithConfigOptions(
//	        config_fxpkg.WithAppConfig(config.AppConfig{...}),
//	        config_fxpkg.WithoutDotEnv(),
//	        config_fxpkg.WithoutConfigFile(),
//	    ),
//	    core.WithLoggerOptions(
//	        logger_fxpkg.WithLoggerConfig(logger.Config{...}),
//	    ),
//	)
func WithConfigOptions(opts ...config_fx_pkg.Option) Option {
	return func(co *coreOptions) {
		co.configOpts = append(co.configOpts, opts...)
	}
}

// WithLoggerOptions passes logger options directly to the underlying logger module.
// Use this to provide static logger config or customize logger behavior.
func WithLoggerOptions(opts ...logger_fx_pkg.Option) Option {
	return func(co *coreOptions) {
		co.loggerOpts = append(co.loggerOpts, opts...)
	}
}

// NewCoreModule provides core functionality: config, logger, and health.
// It also sets increased startup and shutdown timeouts for fx application lifecycle.
//
// Config options are passed via WithConfigOptions.
// Logger options are passed via WithLoggerOptions.
//
// Example usage:
//
//	// Production - loads config from environment/koanf
//	core.NewCoreModule()
//
//	// Testing - with static configs
//	core.NewCoreModule(
//	    core.WithConfigOptions(
//	        config_fxpkg.WithAppConfig(config.AppConfig{...}),
//	        config_fxpkg.WithoutDotEnv(),
//	        config_fxpkg.WithoutConfigFile(),
//	    ),
//	    core.WithLoggerOptions(
//	        logger_fxpkg.WithLoggerConfig(logger.Config{...}),
//	    ),
//	)
func NewCoreModule(opts ...Option) fx.Option {
	cfg := &coreOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return fx.Options(
		fx.StartTimeout(5*time.Minute),
		fx.StopTimeout(5*time.Minute),
		config_fx_pkg.NewConfigModule(cfg.configOpts...),
		logger_fx_pkg.NewZapLoggingModule(cfg.loggerOpts...),
		health_fx_pkg.NewReadinessModule(),
		fx.Invoke(func(lc fx.Lifecycle, log *zap.Logger) {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					log.Info("Application stopping...")
					return nil
				},
			})
		}),
	)
}
