package observability

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	TracingEnabled        bool   `mapstructure:"tracing-enabled"`
	OtelCollectorEndpoint string `mapstructure:"otel-collector-endpoint"`
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("observability").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load otel config: %w", err)
	}
	return cfg, nil
}
