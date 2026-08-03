package fxconfig

import (
	coreconfig "github.com/Sokol111/ecommerce-commons/pkg/core/config"
	config "github.com/Sokol111/ecommerce-commons/pkg/kafka/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewKafkaConfigModule provides Kafka configuration for dependency injection.
// By default, configuration is loaded from koanf.
func NewKafkaConfigModule() fx.Option {
	return fx.Options(
		fx.Provide(provideConfig),
	)
}

func provideConfig(loader *coreconfig.Loader, logger *zap.Logger) (config.Config, error) {
	return coreconfig.Load[config.Config](loader, "kafka", nil)
}
