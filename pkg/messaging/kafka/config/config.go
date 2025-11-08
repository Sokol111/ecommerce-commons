package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Brokers         string               `mapstructure:"brokers"`
	SchemaRegistry  SchemaRegistryConfig `mapstructure:"schema-registry"`
	ConsumersConfig ConsumersConfig      `mapstructure:"consumers-config"`
}

type ConsumersConfig struct {
	GroupID         string           `mapstructure:"group-id"`
	AutoOffsetReset string           `mapstructure:"auto-offset-reset"`
	ConsumerConfig  []ConsumerConfig `mapstructure:"consumers"`
}

type ConsumerConfig struct {
	Name            string `mapstructure:"name"`
	Topic           string `mapstructure:"topic"`
	Subject         string `mapstructure:"subject"`
	GroupID         string `mapstructure:"group-id"`
	AutoOffsetReset string `mapstructure:"auto-offset-reset"`
}

type SchemaRegistryConfig struct {
	URL                 string `mapstructure:"url"`
	CacheCapacity       int    `mapstructure:"cache-capacity"`
	AutoRegisterSchemas bool   `mapstructure:"auto_register_schemas"`
}

func NewKafkaConfigModule() fx.Option {
	return fx.Provide(
		newConfig,
	)
}

func newConfig(v *viper.Viper, logger *zap.Logger) (Config, error) {
	var cfg Config
	if err := v.Sub("kafka").Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load kafka config: %w", err)
	}

	if cfg.SchemaRegistry.CacheCapacity == 0 {
		cfg.SchemaRegistry.CacheCapacity = 1000
	}

	// Apply defaults from global consumer config to individual consumers
	for i := range cfg.ConsumersConfig.ConsumerConfig {
		if cfg.ConsumersConfig.ConsumerConfig[i].GroupID == "" {
			cfg.ConsumersConfig.ConsumerConfig[i].GroupID = cfg.ConsumersConfig.GroupID
		}
		if cfg.ConsumersConfig.ConsumerConfig[i].AutoOffsetReset == "" {
			cfg.ConsumersConfig.ConsumerConfig[i].AutoOffsetReset = cfg.ConsumersConfig.AutoOffsetReset
		}
		// Apply default subject naming convention: {topic}-value
		if cfg.ConsumersConfig.ConsumerConfig[i].Subject == "" {
			cfg.ConsumersConfig.ConsumerConfig[i].Subject = cfg.ConsumersConfig.ConsumerConfig[i].Topic + "-value"
		}
	}

	logger.Info("loaded kafka config", zap.Any("config", cfg))
	return cfg, nil
}
